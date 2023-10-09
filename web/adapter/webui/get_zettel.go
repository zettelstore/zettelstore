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

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/sx.fossil"
	"zettelstore.de/z/box"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/server"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

// MakeGetHTMLZettelHandler creates a new HTTP handler for the use case "get zettel".
func (wui *WebUI) MakeGetHTMLZettelHandler(evaluate *usecase.Evaluate, getZettel usecase.GetZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		path := r.URL.Path[1:]
		zid, err := id.Parse(path)
		if err != nil {
			wui.reportError(ctx, w, box.ErrInvalidZid{Zid: path})
			return
		}

		q := r.URL.Query()
		zn, err := evaluate.Run(ctx, zid, q.Get(api.KeySyntax))
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}

		enc := wui.getSimpleHTMLEncoder()
		metaObj := enc.MetaSxn(zn.InhMeta, createEvalMetadataFunc(ctx, evaluate))
		content, endnotes, err := enc.BlocksSxn(&zn.Ast)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}

		user := server.GetUser(ctx)
		getTextTitle := wui.makeGetTextTitle(ctx, getZettel)

		title := parser.NormalizedSpacedText(zn.InhMeta.GetTitle())
		env, rb := wui.createRenderEnv(ctx, "zettel", wui.rtConfig.Get(ctx, zn.InhMeta, api.KeyLang), title, user)
		rb.bindSymbol(wui.symMetaHeader, metaObj)
		rb.bindString("heading", sx.String(title))
		if role, found := zn.InhMeta.Get(api.KeyRole); found && role != "" {
			rb.bindString("role-url", sx.String(wui.NewURLBuilder('h').AppendQuery(api.KeyRole+api.SearchOperatorHas+role).String()))
		}
		if folgeRole, found := zn.InhMeta.Get(api.KeyFolgeRole); found && folgeRole != "" {
			rb.bindString("folge-role-url", sx.String(wui.NewURLBuilder('h').AppendQuery(api.KeyRole+api.SearchOperatorHas+folgeRole).String()))
		}
		rb.bindString("tag-refs", wui.transformTagSet(api.KeyTags, meta.ListFromValue(zn.InhMeta.GetDefault(api.KeyTags, ""))))
		rb.bindString("predecessor-refs", wui.identifierSetAsLinks(zn.InhMeta, api.KeyPredecessor, getTextTitle))
		rb.bindString("precursor-refs", wui.identifierSetAsLinks(zn.InhMeta, api.KeyPrecursor, getTextTitle))
		rb.bindString("superior-refs", wui.identifierSetAsLinks(zn.InhMeta, api.KeySuperior, getTextTitle))
		rb.bindString("content", content)
		rb.bindString("endnotes", endnotes)
		rb.bindString("folge-links", wui.zettelLinksSxn(zn.InhMeta, api.KeyFolge, getTextTitle))
		rb.bindString("subordinate-links", wui.zettelLinksSxn(zn.InhMeta, api.KeySubordinates, getTextTitle))
		rb.bindString("back-links", wui.zettelLinksSxn(zn.InhMeta, api.KeyBack, getTextTitle))
		rb.bindString("successor-links", wui.zettelLinksSxn(zn.InhMeta, api.KeySuccessors, getTextTitle))
		if role, found := zn.InhMeta.Get(api.KeyRole); found && role != "" {
			for _, part := range []string{"meta", "actions", "heading"} {
				rb.rebindResolved("ROLE-"+role+"-"+part, "ROLE-DEFAULT-"+part)
			}
		}
		wui.bindCommonZettelData(ctx, &rb, user, zn.InhMeta, &zn.Content)
		if rb.err == nil {
			err = wui.renderSxnTemplate(ctx, w, id.ZettelTemplateZid, env)
		} else {
			err = rb.err
		}
		if err != nil {
			wui.reportError(ctx, w, err)
		}
	}
}

func (wui *WebUI) identifierSetAsLinks(m *meta.Meta, key string, getTextTitle getTextTitleFunc) *sx.Pair {
	if values, ok := m.GetList(key); ok {
		return wui.transformIdentifierSet(values, getTextTitle)
	}
	return nil
}

func (wui *WebUI) zettelLinksSxn(m *meta.Meta, key string, getTextTitle getTextTitleFunc) *sx.Pair {
	values, ok := m.GetList(key)
	if !ok || len(values) == 0 {
		return nil
	}
	return wui.zidLinksSxn(values, getTextTitle)
}

func (wui *WebUI) zidLinksSxn(values []string, getTextTitle getTextTitleFunc) (lst *sx.Pair) {
	for i := len(values) - 1; i >= 0; i-- {
		val := values[i]
		zid, err := id.Parse(val)
		if err != nil {
			continue
		}
		if title, found := getTextTitle(zid); found > 0 {
			url := sx.String(wui.NewURLBuilder('h').SetZid(api.ZettelID(zid.String())).String())
			if title == "" {
				lst = lst.Cons(sx.Cons(sx.String(val), url))
			} else {
				lst = lst.Cons(sx.Cons(sx.String(title), url))
			}
		}
	}
	return lst
}
