//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package webui

import (
	"bytes"
	"errors"
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/evaluator"
	"zettelstore.de/z/strfun"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetHTMLZettelHandler creates a new HTTP handler for the use case "get zettel".
func (wui *WebUI) MakeGetHTMLZettelHandler(evaluate *usecase.Evaluate, getMeta usecase.GetMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			wui.reportError(ctx, w, box.ErrNotFound)
			return
		}

		q := r.URL.Query()
		env := evaluator.Environment{
			GetTagRef: func(s string) *ast.Reference {
				return adapter.CreateTagReference(wui, 'h', api.EncodingHTML, s)
			},
			GetHostedRef: func(s string) *ast.Reference {
				return adapter.CreateHostedReference(wui, s)
			},
			GetFoundRef: func(zid id.Zid, fragment string) *ast.Reference {
				return adapter.CreateFoundReference(wui, 'h', "", "", zid, fragment)
			},
			GetImageMaterial: func(zettel domain.Zettel, _ string) ast.InlineEmbedNode {
				return wui.createImageMaterial(zettel.Meta.Zid)
			},
		}
		zn, err := evaluate.Run(ctx, zid, q.Get(api.KeySyntax), &env)

		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}

		evalMeta := func(value string) ast.InlineListNode {
			return evaluate.RunMetadata(ctx, value, &env)
		}
		lang := config.GetLang(zn.InhMeta, wui.rtConfig)
		envHTML := encoder.Environment{
			Lang:           lang,
			Xhtml:          false,
			MarkerExternal: wui.rtConfig.GetMarkerExternal(),
			NewWindow:      true,
			IgnoreMeta:     strfun.NewSet(api.KeyTitle, api.KeyLang),
		}
		metaHeader, err := encodeMeta(zn.InhMeta, evalMeta, api.EncoderHTML, &envHTML)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		textTitle := wui.encodeTitleAsText(ctx, zn.InhMeta, evaluate)
		htmlTitle := wui.encodeTitleAsHTML(ctx, zn.InhMeta, evaluate, &env, &envHTML)
		htmlContent, err := encodeBlocks(zn.Ast, api.EncoderHTML, &envHTML)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		user := wui.getUser(ctx)
		roleText := zn.Meta.GetDefault(api.KeyRole, "*")
		tags := wui.buildTagInfos(zn.Meta)
		canCreate := wui.canCreate(ctx, user)
		getTextTitle := wui.makeGetTextTitle(ctx, getMeta, evaluate)
		extURL, hasExtURL := zn.Meta.Get(api.KeyURL)
		folgeLinks := wui.encodeZettelLinks(zn.InhMeta, api.KeyFolge, getTextTitle)
		backLinks := wui.encodeZettelLinks(zn.InhMeta, api.KeyBack, getTextTitle)
		apiZid := api.ZettelID(zid.String())
		var base baseData
		wui.makeBaseData(ctx, lang, textTitle, user, &base)
		base.MetaHeader = metaHeader
		wui.renderTemplate(ctx, w, id.ZettelTemplateZid, &base, struct {
			HTMLTitle     string
			CanWrite      bool
			EditURL       string
			Zid           string
			InfoURL       string
			RoleText      string
			RoleURL       string
			HasTags       bool
			Tags          []simpleLink
			CanCopy       bool
			CopyURL       string
			CanFolge      bool
			FolgeURL      string
			PrecursorRefs string
			HasExtURL     bool
			ExtURL        string
			ExtNewWindow  string
			Content       string
			HasFolgeLinks bool
			FolgeLinks    []simpleLink
			HasBackLinks  bool
			BackLinks     []simpleLink
		}{
			HTMLTitle:     htmlTitle,
			CanWrite:      wui.canWrite(ctx, user, zn.Meta, zn.Content),
			EditURL:       wui.NewURLBuilder('e').SetZid(apiZid).String(),
			Zid:           zid.String(),
			InfoURL:       wui.NewURLBuilder('i').SetZid(apiZid).String(),
			RoleText:      roleText,
			RoleURL:       wui.NewURLBuilder('h').AppendQuery("role", roleText).String(),
			HasTags:       len(tags) > 0,
			Tags:          tags,
			CanCopy:       canCreate && !zn.Content.IsBinary(),
			CopyURL:       wui.NewURLBuilder('c').SetZid(apiZid).String(),
			CanFolge:      canCreate,
			FolgeURL:      wui.NewURLBuilder('f').SetZid(apiZid).String(),
			PrecursorRefs: wui.encodeIdentifierSet(zn.InhMeta, api.KeyPrecursor, getTextTitle),
			ExtURL:        extURL,
			HasExtURL:     hasExtURL,
			ExtNewWindow:  htmlAttrNewWindow(envHTML.NewWindow && hasExtURL),
			Content:       htmlContent,
			HasFolgeLinks: len(folgeLinks) > 0,
			FolgeLinks:    folgeLinks,
			HasBackLinks:  len(backLinks) > 0,
			BackLinks:     backLinks,
		})
	}
}

// errNoSuchEncoding signals an unsupported encoding encoding
var errNoSuchEncoding = errors.New("no such encoding")

// encodeInlines returns a string representation of the inline slice.
func encodeInlines(is *ast.InlineListNode, enc api.EncodingEnum, env *encoder.Environment) (string, error) {
	if is == nil {
		return "", nil
	}
	encdr := encoder.Create(enc, env)
	if encdr == nil {
		return "", errNoSuchEncoding
	}

	var buf bytes.Buffer
	_, err := encdr.WriteInlines(&buf, is)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func encodeBlocks(bln *ast.BlockListNode, enc api.EncodingEnum, env *encoder.Environment) (string, error) {
	encdr := encoder.Create(enc, env)
	if encdr == nil {
		return "", errNoSuchEncoding
	}

	var buf bytes.Buffer
	_, err := encdr.WriteBlocks(&buf, bln)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func encodeMeta(
	m *meta.Meta, evalMeta encoder.EvalMetaFunc,
	enc api.EncodingEnum, env *encoder.Environment,
) (string, error) {
	encdr := encoder.Create(enc, env)
	if encdr == nil {
		return "", errNoSuchEncoding
	}

	var buf bytes.Buffer
	_, err := encdr.WriteMeta(&buf, m, evalMeta)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (wui *WebUI) buildTagInfos(m *meta.Meta) []simpleLink {
	var tagInfos []simpleLink
	if tags, ok := m.GetList(api.KeyTags); ok {
		ub := wui.NewURLBuilder('h')
		tagInfos = make([]simpleLink, len(tags))
		for i, tag := range tags {
			tagInfos[i] = simpleLink{Text: tag, URL: ub.AppendQuery(api.KeyAllTags, tag).String()}
			ub.ClearQuery()
		}
	}
	return tagInfos
}

func (wui *WebUI) encodeIdentifierSet(m *meta.Meta, key string, getTextTitle getTextTitleFunc) string {
	if value, ok := m.Get(key); ok {
		var buf bytes.Buffer
		wui.writeIdentifierSet(&buf, meta.ListFromValue(value), getTextTitle)
		return buf.String()
	}
	return ""
}

func (wui *WebUI) encodeZettelLinks(m *meta.Meta, key string, getTextTitle getTextTitleFunc) []simpleLink {
	values, ok := m.GetList(key)
	if !ok || len(values) == 0 {
		return nil
	}
	return wui.encodeZidLinks(values, getTextTitle)
}

func (wui *WebUI) encodeZidLinks(values []string, getTextTitle getTextTitleFunc) []simpleLink {
	result := make([]simpleLink, 0, len(values))
	for _, val := range values {
		zid, err := id.Parse(val)
		if err != nil {
			continue
		}
		if title, found := getTextTitle(zid); found > 0 {
			url := wui.NewURLBuilder('h').SetZid(api.ZettelID(zid.String())).String()
			if title == "" {
				result = append(result, simpleLink{Text: val, URL: url})
			} else {
				result = append(result, simpleLink{Text: title, URL: url})
			}
		}
	}
	return result
}
