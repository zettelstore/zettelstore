//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package webui provides web-UI handlers for web requests.
package webui

import (
	"bytes"
	"errors"
	"net/http"
	"strings"

	"zettelstore.de/z/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/encoder/encfun"
	"zettelstore.de/z/evaluator"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetHTMLZettelHandler creates a new HTTP handler for the use case "get zettel".
func (wui *WebUI) MakeGetHTMLZettelHandler(evaluateZettel usecase.EvaluateZettel, getMeta usecase.GetMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			wui.reportError(ctx, w, box.ErrNotFound)
			return
		}

		q := r.URL.Query()
		zn, err := evaluateZettel.Run(ctx, zid, &evaluator.Environment{
			Syntax: q.Get(meta.KeySyntax),
			Config: wui.rtConfig,
			GetHostedRef: func(s string) *ast.Reference {
				return adapter.CreateHostedReference(wui, s)
			},
			GetFoundRef: func(zid id.Zid, fragment string) *ast.Reference {
				return adapter.CreateFoundReference(wui, 'h', "", "", zid, fragment)
			},
			GetImageRef: func(zid id.Zid, state ast.RefState) *ast.Reference {
				return adapter.CreateImageReference(wui, zid, state)
			},
		})

		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}

		lang := config.GetLang(zn.InhMeta, wui.rtConfig)
		envHTML := encoder.Environment{
			Lang:           lang,
			Xhtml:          false,
			MarkerExternal: wui.rtConfig.GetMarkerExternal(),
			NewWindow:      true,
			IgnoreMeta:     map[string]bool{meta.KeyTitle: true, meta.KeyLang: true},
		}
		metaHeader, err := encodeMeta(zn.InhMeta, api.EncoderHTML, &envHTML)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		htmlTitle, err := encodeInlines(
			encfun.MetaAsInlineList(zn.InhMeta, meta.KeyTitle), api.EncoderHTML, &envHTML)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		htmlContent, err := encodeBlocks(zn.Ast, api.EncoderHTML, &envHTML)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		textTitle := encfun.MetaAsText(zn.InhMeta, meta.KeyTitle)
		user := wui.getUser(ctx)
		roleText := zn.Meta.GetDefault(meta.KeyRole, "*")
		tags := wui.buildTagInfos(zn.Meta)
		canCreate := wui.canCreate(ctx, user)
		getTitle := makeGetTitle(ctx, getMeta, &encoder.Environment{Lang: lang})
		extURL, hasExtURL := zn.Meta.Get(meta.KeyURL)
		folgeLinks := wui.encodeZettelLinks(zn.InhMeta, meta.KeyFolge, getTitle)
		backLinks := wui.encodeZettelLinks(zn.InhMeta, meta.KeyBack, getTitle)
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
			EditURL:       wui.NewURLBuilder('e').SetZid(zid).String(),
			Zid:           zid.String(),
			InfoURL:       wui.NewURLBuilder('i').SetZid(zid).String(),
			RoleText:      roleText,
			RoleURL:       wui.NewURLBuilder('h').AppendQuery("role", roleText).String(),
			HasTags:       len(tags) > 0,
			Tags:          tags,
			CanCopy:       canCreate && !zn.Content.IsBinary(),
			CopyURL:       wui.NewURLBuilder('c').SetZid(zid).String(),
			CanFolge:      canCreate,
			FolgeURL:      wui.NewURLBuilder('f').SetZid(zid).String(),
			PrecursorRefs: wui.encodeMetaKey(zn.InhMeta, meta.KeyPrecursor, getTitle),
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

	var content strings.Builder
	_, err := encdr.WriteInlines(&content, is)
	if err != nil {
		return "", err
	}
	return content.String(), nil
}

func encodeBlocks(bln *ast.BlockListNode, enc api.EncodingEnum, env *encoder.Environment) (string, error) {
	encdr := encoder.Create(enc, env)
	if encdr == nil {
		return "", errNoSuchEncoding
	}

	var content strings.Builder
	_, err := encdr.WriteBlocks(&content, bln)
	if err != nil {
		return "", err
	}
	return content.String(), nil
}

func encodeMeta(m *meta.Meta, enc api.EncodingEnum, env *encoder.Environment) (string, error) {
	encdr := encoder.Create(enc, env)
	if encdr == nil {
		return "", errNoSuchEncoding
	}

	var content strings.Builder
	_, err := encdr.WriteMeta(&content, m)
	if err != nil {
		return "", err
	}
	return content.String(), nil
}

func (wui *WebUI) buildTagInfos(m *meta.Meta) []simpleLink {
	var tagInfos []simpleLink
	if tags, ok := m.GetList(meta.KeyTags); ok {
		ub := wui.NewURLBuilder('h')
		tagInfos = make([]simpleLink, len(tags))
		for i, tag := range tags {
			tagInfos[i] = simpleLink{Text: tag, URL: ub.AppendQuery("tags", tag).String()}
			ub.ClearQuery()
		}
	}
	return tagInfos
}

func (wui *WebUI) encodeMetaKey(m *meta.Meta, key string, getTitle getTitleFunc) string {
	if _, ok := m.Get(key); ok {
		var buf bytes.Buffer
		wui.writeHTMLMetaValue(&buf, m, key, getTitle, nil)
		return buf.String()
	}
	return ""
}

func (wui *WebUI) encodeZettelLinks(m *meta.Meta, key string, getTitle getTitleFunc) []simpleLink {
	values, ok := m.GetList(key)
	if !ok || len(values) == 0 {
		return nil
	}
	result := make([]simpleLink, 0, len(values))
	for _, val := range values {
		zid, err := id.Parse(val)
		if err != nil {
			continue
		}
		if title, found := getTitle(zid, api.EncoderText); found > 0 {
			url := wui.NewURLBuilder('h').SetZid(zid).String()
			if title == "" {
				result = append(result, simpleLink{Text: val, URL: url})
			} else {
				result = append(result, simpleLink{Text: title, URL: url})
			}
		}
	}
	return result
}
