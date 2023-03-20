//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package webui

import (
	"net/http"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/parser"
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

		enc := wui.getSimpleHTMLEncoder()
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
		roleText := zn.Meta.GetDefault(api.KeyRole, "")
		canCreate := wui.canCreate(ctx, user)
		getTextTitle := wui.makeGetTextTitle(ctx, getMeta)
		extURL, hasExtURL := wui.formatURLFromMeta(zn.Meta, api.KeyURL)
		folgeLinks := createSimpleLinks(wui.encodeZettelLinks(zn.InhMeta, api.KeyFolge, getTextTitle))
		backLinks := createSimpleLinks(wui.encodeZettelLinks(zn.InhMeta, api.KeyBack, getTextTitle))
		successorLinks := createSimpleLinks(wui.encodeZettelLinks(zn.InhMeta, api.KeySuccessors, getTextTitle))
		apiZid := api.ZettelID(zid.String())

		var base baseData
		title := parser.NormalizedSpacedText(zn.InhMeta.GetTitle())
		wui.makeBaseData(ctx, wui.rtConfig.Get(ctx, zn.InhMeta, api.KeyLang), title, roleCSSURL, user, &base)
		base.MetaHeader = enc.MetaString(zn.InhMeta, createEvalMetadataFunc(ctx, evaluate))
		wui.renderTemplate(ctx, w, id.ZettelTemplateZid, &base, struct {
			Heading         string
			RoleCSS         string
			CanWrite        bool
			EditURL         string
			Zid             string
			InfoURL         string
			RoleText        string
			RoleURL         string
			Tags            simpleLinks
			CanCopy         bool
			CopyURL         string
			CanVersion      bool
			VersionURL      string
			CanFolge        bool
			FolgeURL        string
			PredecessorRefs string
			PrecursorRefs   string
			HasExtURL       bool
			ExtURL          string
			Author          string
			Content         string
			NeedBottomNav   bool
			FolgeLinks      simpleLinks
			BackLinks       simpleLinks
			SuccessorLinks  simpleLinks
		}{
			Heading:         title,
			RoleCSS:         roleCSSURL,
			CanWrite:        wui.canWrite(ctx, user, zn.Meta, zn.Content),
			EditURL:         wui.NewURLBuilder('e').SetZid(apiZid).String(),
			Zid:             zid.String(),
			InfoURL:         wui.NewURLBuilder('i').SetZid(apiZid).String(),
			RoleText:        roleText,
			RoleURL:         wui.NewURLBuilder('h').AppendQuery(api.KeyRole + api.SearchOperatorHas + roleText).String(),
			Tags:            createSimpleLinks(wui.buildTagInfos(zn.Meta)),
			CanCopy:         canCreate && !zn.Content.IsBinary(),
			CopyURL:         wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionCopy).String(),
			CanVersion:      canCreate,
			VersionURL:      wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionVersion).String(),
			CanFolge:        canCreate,
			FolgeURL:        wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionFolge).String(),
			PredecessorRefs: wui.encodeIdentifierSet(zn.InhMeta, api.KeyPredecessor, getTextTitle),
			PrecursorRefs:   wui.encodeIdentifierSet(zn.InhMeta, api.KeyPrecursor, getTextTitle),
			ExtURL:          extURL,
			HasExtURL:       hasExtURL,
			Author:          zn.Meta.GetDefault(api.KeyAuthor, ""),
			Content:         htmlContent,
			NeedBottomNav:   folgeLinks.Has || backLinks.Has || successorLinks.Has,
			FolgeLinks:      folgeLinks,
			BackLinks:       backLinks,
			SuccessorLinks:  successorLinks,
		})
	}
}

func (wui *WebUI) formatURLFromMeta(m *meta.Meta, key string) (string, bool) {
	val, found := m.Get(key)
	if !found {
		return "", false
	}
	if val == "" {
		return "", false
	}

	var sb strings.Builder
	_, err := wui.htmlGen.WriteHTML(&sb, wui.transformURL(val))
	if err != nil {
		return "", false
	}
	return sb.String(), true
}

func (wui *WebUI) buildTagInfos(m *meta.Meta) []simpleLink {
	var tagInfos []simpleLink
	if tags, ok := m.GetList(api.KeyTags); ok {
		ub := wui.NewURLBuilder('h')
		tagInfos = make([]simpleLink, len(tags))
		for i, tag := range tags {
			tagInfos[i] = simpleLink{Text: tag, URL: ub.AppendQuery(api.KeyTags + api.SearchOperatorHas + tag).String()}
			ub.ClearQuery()
		}
	}
	return tagInfos
}

func (wui *WebUI) encodeIdentifierSet(m *meta.Meta, key string, getTextTitle getTextTitleFunc) string {
	if value, ok := m.Get(key); ok {
		sval := wui.transformIdentifierSet(meta.ListFromValue(value), getTextTitle)
		var sb strings.Builder
		wui.htmlGen.WriteHTML(&sb, sval)
		return sb.String()
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
