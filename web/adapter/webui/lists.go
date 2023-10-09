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
	"net/url"
	"slices"
	"strconv"
	"strings"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/sx.fossil"
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
func (wui *WebUI) MakeListHTMLMetaHandler(queryMeta *usecase.Query, tagZettel *usecase.TagZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		urlQuery := r.URL.Query()
		if wui.handleTagZettel(w, r, tagZettel, urlQuery) {
			return
		}
		q := adapter.GetQuery(urlQuery)
		q = q.SetDeterministic()
		ctx := r.Context()
		metaSeq, err := queryMeta.Run(ctx, q)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		if actions := q.Actions(); len(actions) > 0 {
			switch actions[0] {
			case "ATOM":
				wui.renderAtom(w, q, metaSeq)
				return
			case "RSS":
				wui.renderRSS(ctx, w, q, metaSeq)
				return
			}
		}
		var content, endnotes *sx.Pair
		if bn := evaluator.QueryAction(ctx, q, metaSeq, wui.rtConfig); bn != nil {
			enc := wui.getSimpleHTMLEncoder()
			content, endnotes, err = enc.BlocksSxn(&ast.BlockSlice{bn})
			if err != nil {
				wui.reportError(ctx, w, err)
				return
			}
		}

		user := server.GetUser(ctx)
		env, rb := wui.createRenderEnv(
			ctx, "list",
			wui.rtConfig.Get(ctx, nil, api.KeyLang),
			wui.rtConfig.GetSiteName(), user)
		if q == nil {
			rb.bindString("heading", sx.String(wui.rtConfig.GetSiteName()))
		} else {
			var sb strings.Builder
			q.PrintHuman(&sb)
			rb.bindString("heading", sx.String(sb.String()))
		}
		rb.bindString("query-value", sx.String(q.String()))
		if tzl := q.GetMetaValues(api.KeyTags); len(tzl) > 0 {
			sxTzl, sxNoTzl := wui.transformTagZettelList(ctx, tagZettel, tzl)
			if !sx.IsNil(sxTzl) {
				rb.bindString("tag-zettel", sxTzl)
			}
			if !sx.IsNil(sxNoTzl) {
				rb.bindString("create-tag-zettel", sxNoTzl)
			}
		}
		rb.bindString("content", content)
		rb.bindString("endnotes", endnotes)
		apiURL := wui.NewURLBuilder('z').AppendQuery(q.String())
		seed, found := q.GetSeed()
		if found {
			apiURL = apiURL.AppendKVQuery(api.QueryKeySeed, strconv.Itoa(seed))
		} else {
			seed = 0
		}
		rb.bindString("plain-url", sx.String(apiURL.String()))
		rb.bindString("data-url", sx.String(apiURL.AppendKVQuery(api.QueryKeyEncoding, api.EncodingData).String()))
		if wui.canCreate(ctx, user) {
			rb.bindString("create-url", sx.String(wui.createNewURL))
			rb.bindString("seed", sx.Int64(seed))
		}
		if rb.err == nil {
			err = wui.renderSxnTemplate(ctx, w, id.ListTemplateZid, env)
		} else {
			err = rb.err
		}
		if err != nil {
			wui.reportError(ctx, w, err)
		}
	}
}

func (wui *WebUI) transformTagZettelList(ctx context.Context, tagZettel *usecase.TagZettel, tags []string) (withZettel, withoutZettel *sx.Pair) {
	slices.Reverse(tags)
	for _, tag := range tags {
		if _, err := tagZettel.Run(ctx, tag); err == nil {
			u := wui.NewURLBuilder('h').AppendKVQuery(api.QueryKeyTag, tag)
			withZettel = wui.prependTagZettel(withZettel, tag, u)
		} else {
			u := wui.NewURLBuilder('c').SetZid(api.ZidTemplateNewTag).AppendKVQuery(queryKeyAction, valueActionNew).AppendKVQuery(api.KeyTitle, tag)
			withoutZettel = wui.prependTagZettel(withoutZettel, tag, u)
		}
	}
	return withZettel, withoutZettel
}

func (wui *WebUI) prependTagZettel(sxZtl *sx.Pair, tag string, u *api.URLBuilder) *sx.Pair {
	link := sx.MakeList(
		wui.symA,
		sx.MakeList(
			wui.symAttr,
			sx.Cons(wui.symHref, sx.String(u.String())),
		),
		sx.String(tag),
	)
	if sxZtl != nil {
		sxZtl = sxZtl.Cons(sx.String(", "))
	}
	return sxZtl.Cons(link)
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

func (wui *WebUI) handleTagZettel(w http.ResponseWriter, r *http.Request, tagZettel *usecase.TagZettel, vals url.Values) bool {
	tag := vals.Get(api.QueryKeyTag)
	if tag == "" {
		return false
	}
	ctx := r.Context()
	z, err := tagZettel.Run(ctx, tag)
	if err != nil {
		wui.reportError(ctx, w, err)
		return true
	}
	wui.redirectFound(w, r, wui.NewURLBuilder('h').SetZid(api.ZettelID(z.Meta.Zid.String())))
	return true
}
