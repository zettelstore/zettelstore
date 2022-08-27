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
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder/textenc"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/server"
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
		zn, err := evaluate.Run(ctx, zid, q.Get(api.KeySyntax))

		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}

		enc := wui.createZettelEncoder(ctx, zn.InhMeta)
		evalMetadata := createEvalMetadataFunc(ctx, evaluate)
		textTitle := encodeEvaluatedTitleText(zn.InhMeta, evalMetadata, wui.gentext)
		htmlTitle := encodeEvaluatedTitleHTML(zn.InhMeta, evalMetadata, enc)
		htmlContent, err := enc.BlocksString(&zn.Ast)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		var roleCSSURL string
		cssZid, err := wui.retrieveCSSZidFromRole(ctx, *zn.InhMeta)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		if cssZid != id.Invalid {
			roleCSSURL = wui.NewURLBuilder('z').SetZid(api.ZettelID(cssZid.String())).String()
		}
		user := server.GetUser(ctx)
		roleText := zn.Meta.GetDefault(api.KeyRole, "*")
		tags := wui.buildTagInfos(zn.Meta)
		canCreate := wui.canCreate(ctx, user)
		getTextTitle := wui.makeGetTextTitle(createGetMetadataFunc(ctx, getMeta), evalMetadata)
		extURL, hasExtURL := zn.Meta.Get(api.KeyURL)
		folgeLinks := wui.encodeZettelLinks(zn.InhMeta, api.KeyFolge, getTextTitle)
		backLinks := wui.encodeZettelLinks(zn.InhMeta, api.KeyBack, getTextTitle)
		apiZid := api.ZettelID(zid.String())
		var base baseData
		wui.makeBaseData(ctx, wui.rtConfig.Get(ctx, zn.InhMeta, api.KeyLang), textTitle, roleCSSURL, user, &base)
		base.MetaHeader = enc.MetaString(zn.InhMeta, evalMetadata)
		wui.renderTemplate(ctx, w, id.ZettelTemplateZid, &base, struct {
			HTMLTitle     string
			RoleCSS       string
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
			RoleCSS:       roleCSSURL,
			CanWrite:      wui.canWrite(ctx, user, zn.Meta, zn.Content),
			EditURL:       wui.NewURLBuilder('e').SetZid(apiZid).String(),
			Zid:           zid.String(),
			InfoURL:       wui.NewURLBuilder('i').SetZid(apiZid).String(),
			RoleText:      roleText,
			RoleURL:       wui.NewURLBuilder('h').AppendSearch("role:" + roleText).String(),
			HasTags:       len(tags) > 0,
			Tags:          tags,
			CanCopy:       canCreate && !zn.Content.IsBinary(),
			CopyURL:       wui.NewURLBuilder('c').SetZid(apiZid).AppendQuery(queryKeyAction, valueActionCopy).String(),
			CanFolge:      canCreate,
			FolgeURL:      wui.NewURLBuilder('c').SetZid(apiZid).AppendQuery(queryKeyAction, valueActionFolge).String(),
			PrecursorRefs: wui.encodeIdentifierSet(zn.InhMeta, api.KeyPrecursor, getTextTitle),
			ExtURL:        extURL,
			HasExtURL:     hasExtURL,
			ExtNewWindow:  htmlAttrNewWindow(hasExtURL),
			Content:       htmlContent,
			HasFolgeLinks: len(folgeLinks) > 0,
			FolgeLinks:    folgeLinks,
			HasBackLinks:  len(backLinks) > 0,
			BackLinks:     backLinks,
		})
	}
}

func encodeInlinesText(is *ast.InlineSlice, enc *textenc.Encoder) (string, error) {
	if is == nil || len(*is) == 0 {
		return "", nil
	}

	var buf bytes.Buffer
	_, err := enc.WriteInlines(&buf, is)
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
			tagInfos[i] = simpleLink{Text: tag, URL: ub.AppendSearch(api.KeyAllTags + api.SearchOperatorHas + tag).String()}
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
