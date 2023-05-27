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
	"context"
	"io"
	"net/http"
	"strings"

	"codeberg.org/t73fde/sxpf"
	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/encoding/atom"
	"zettelstore.de/z/encoding/rss"
	"zettelstore.de/z/encoding/xml"
	"zettelstore.de/z/evaluator"
	"zettelstore.de/z/query"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/server"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

// MakeListHTMLMetaHandler creates a HTTP handler for rendering the list of zettel as HTML.
func (wui *WebUI) MakeListHTMLMetaHandler(listMeta usecase.ListMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := adapter.GetQuery(r.URL.Query())
		q = q.SetDeterministic()
		ctx := r.Context()
		metaList, err := listMeta.Run(ctx, q)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		if actions := q.Actions(); len(actions) > 0 {
			switch actions[0] {
			case "ATOM":
				wui.renderAtom(w, q, metaList)
				return
			case "RSS":
				wui.renderRSS(ctx, w, q, metaList)
				return
			}
		}
		var content, endnotes *sxpf.List
		if bn := evaluator.QueryAction(ctx, q, metaList, wui.rtConfig); bn != nil {
			enc := wui.getSimpleHTMLEncoder()
			content, endnotes, err = enc.BlocksSxn(&ast.BlockSlice{bn})
			if err != nil {
				wui.reportError(ctx, w, err)
				return
			}
		}
		seed, found := q.GetSeed()
		if !found {
			seed = 0
		}

		user := server.GetUser(ctx)
		env, err := wui.createRenderEnv(
			ctx, "list",
			wui.rtConfig.Get(ctx, nil, api.KeyLang),
			wui.rtConfig.GetSiteName(), user)
		rb := makeRenderBinder(wui.sf, env, err)
		if q == nil {
			rb.bindString("heading", sxpf.MakeString(wui.rtConfig.GetSiteName()))
		} else {
			var sb strings.Builder
			q.PrintHuman(&sb)
			rb.bindString("heading", sxpf.MakeString(sb.String()))
		}
		rb.bindString("query-value", sxpf.MakeString(q.String()))
		rb.bindString("content", content)
		rb.bindString("endnotes", endnotes)
		rb.bindString("can-create", sxpf.MakeBoolean(wui.canCreate(ctx, user)))
		rb.bindString("create-url", sxpf.MakeString(wui.createNewURL))
		rb.bindString("seed", sxpf.Int64(seed))
		if rb.err == nil {
			err = wui.renderSxnTemplate(ctx, w, id.ListTemplateZid, env)
		}
		if err != nil {
			wui.reportError(ctx, w, err)
		}
	}
}

func (wui *WebUI) renderRSS(ctx context.Context, w http.ResponseWriter, q *query.Query, ml []*meta.Meta) {
	var rssConfig rss.Configuration
	rssConfig.Setup(ctx, wui.rtConfig)
	if actions := q.Actions(); len(actions) > 2 && actions[1] == "TITLE" {
		rssConfig.Title = strings.Join(actions[2:], " ")
	}
	data := rssConfig.Marshal(q, ml)

	adapter.PrepareHeader(w, rss.ContentType)
	w.WriteHeader(http.StatusOK)
	var err error
	if _, err = io.WriteString(w, xml.Header); err == nil {
		_, err = w.Write(data)
	}
	if err != nil {
		wui.log.IfErr(err).Msg("unable to write RSS data")
	}
}

func (wui *WebUI) renderAtom(w http.ResponseWriter, q *query.Query, ml []*meta.Meta) {
	var atomConfig atom.Configuration
	atomConfig.Setup(wui.rtConfig)
	if actions := q.Actions(); len(actions) > 2 && actions[1] == "TITLE" {
		atomConfig.Title = strings.Join(actions[2:], " ")
	}
	data := atomConfig.Marshal(q, ml)

	adapter.PrepareHeader(w, atom.ContentType)
	w.WriteHeader(http.StatusOK)
	var err error
	if _, err = io.WriteString(w, xml.Header); err == nil {
		_, err = w.Write(data)
	}
	if err != nil {
		wui.log.IfErr(err).Msg("unable to write Atom data")
	}
}
