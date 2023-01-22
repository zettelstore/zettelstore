//-----------------------------------------------------------------------------
// Copyright (c) 2020-2023 Detlef Stern
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
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoding/atom"
	"zettelstore.de/z/encoding/rss"
	"zettelstore.de/z/encoding/xml"
	"zettelstore.de/z/evaluator"
	"zettelstore.de/z/query"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/server"
)

// MakeListHTMLMetaHandler creates a HTTP handler for rendering the list of zettel as HTML.
func (wui *WebUI) MakeListHTMLMetaHandler(listMeta usecase.ListMeta, evaluate *usecase.Evaluate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := adapter.GetQuery(r.URL.Query())
		ctx := r.Context()
		metaList, err := listMeta.Run(box.NoEnrichQuery(ctx, q), q)
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
		bns := evaluate.RunBlockNode(ctx, evaluator.QueryAction(ctx, q, metaList, wui.rtConfig))
		enc := wui.getSimpleHTMLEncoder()
		htmlContent, err := enc.BlocksString(&bns)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		seed, found := q.GetSeed()
		if !found {
			seed = 0
		}
		user := server.GetUser(ctx)
		var base baseData
		wui.makeBaseData(ctx, wui.rtConfig.Get(ctx, nil, api.KeyLang), wui.rtConfig.GetSiteName(), "", user, &base)
		wui.renderTemplate(ctx, w, id.ListTemplateZid, &base, struct {
			Title         string
			SearchURL     string
			QueryValue    string
			QueryKeyQuery string
			Content       string
			CreateURL     string
			QueryKeySeed  string
			Seed          int
		}{
			Title:         wui.listTitleQuery(q),
			SearchURL:     base.SearchURL,
			QueryValue:    q.String(),
			QueryKeyQuery: base.QueryKeyQuery,
			Content:       htmlContent,
			CreateURL:     base.CreateNewURL,
			QueryKeySeed:  base.QueryKeySeed,
			Seed:          seed,
		})
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

// MakeZettelContextHandler creates a new HTTP handler for the use case "zettel context".
func (wui *WebUI) MakeZettelContextHandler(getContext usecase.ZettelContext, evaluate *usecase.Evaluate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			wui.reportError(ctx, w, box.ErrNotFound)
			return
		}
		q := r.URL.Query()
		dir := adapter.GetZCDirection(q.Get(api.QueryKeyDir))
		cost := getIntParameter(q, api.QueryKeyCost, 17)
		limit := getIntParameter(q, api.QueryKeyLimit, 200)
		metaList, err := getContext.Run(ctx, zid, dir, cost, limit)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		bns := evaluate.RunBlockNode(ctx, evaluator.QueryAction(ctx, nil, metaList, wui.rtConfig))
		enc := wui.getSimpleHTMLEncoder()
		htmlContent, err := enc.BlocksString(&bns)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		apiZid := api.ZettelID(zid.String())
		costs := []string{"2", "3", "5", "8", "13", "21", "34", "55", "89"}
		costLinks := make([]simpleLink, len(costs))
		costURL := wui.NewURLBuilder('k').SetZid(apiZid)
		for i, cost := range costs {
			costURL.ClearQuery()
			switch dir {
			case usecase.ZettelContextBackward:
				costURL.AppendKVQuery(api.QueryKeyDir, api.DirBackward)
			case usecase.ZettelContextForward:
				costURL.AppendKVQuery(api.QueryKeyDir, api.DirForward)
			}
			costURL.AppendKVQuery(api.QueryKeyCost, cost)
			costLinks[i].Text = cost
			costLinks[i].URL = costURL.String()
		}
		var base baseData
		user := server.GetUser(ctx)
		wui.makeBaseData(ctx, wui.rtConfig.Get(ctx, nil, api.KeyLang), wui.rtConfig.GetSiteName(), "", user, &base)
		wui.renderTemplate(ctx, w, id.ContextTemplateZid, &base, struct {
			Title   string
			InfoURL string
			Costs   []simpleLink
			Content string
		}{
			Title:   "Zettel Context",
			InfoURL: wui.NewURLBuilder('i').SetZid(apiZid).String(),
			Costs:   costLinks,
			Content: htmlContent,
		})
	}
}

func getIntParameter(q url.Values, key string, minValue int) int {
	val, ok := adapter.GetInteger(q, key)
	if !ok || val < 0 {
		return minValue
	}
	return val
}

func (wui *WebUI) listTitleQuery(q *query.Query) string {
	if q == nil {
		return wui.rtConfig.GetSiteName()
	}
	var buf bytes.Buffer
	q.PrintHuman(&buf)
	return buf.String()
}
