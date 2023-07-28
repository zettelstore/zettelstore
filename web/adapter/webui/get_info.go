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
	"net/http"
	"sort"
	"strings"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/sx.fossil"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/collect"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/evaluator"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/query"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/server"
	"zettelstore.de/z/zettel/id"
)

// MakeGetInfoHandler creates a new HTTP handler for the use case "get zettel".
func (wui *WebUI) MakeGetInfoHandler(
	parseZettel usecase.ParseZettel,
	evaluate *usecase.Evaluate,
	getZettel usecase.GetZettel,
	getAllMeta usecase.GetAllZettel,
	unlinkedRefs usecase.UnlinkedReferences,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		q := r.URL.Query()

		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			wui.reportError(ctx, w, box.ErrNotFound)
			return
		}

		zn, err := parseZettel.Run(ctx, zid, q.Get(api.KeySyntax))
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}

		enc := wui.getSimpleHTMLEncoder()
		getTextTitle := wui.makeGetTextTitle(ctx, getZettel)
		evalMeta := func(val string) ast.InlineSlice {
			return evaluate.RunMetadata(ctx, val)
		}
		pairs := zn.Meta.ComputedPairs()
		metadata := sx.Nil()
		for i := len(pairs) - 1; i >= 0; i-- {
			key := pairs[i].Key
			sxval := wui.writeHTMLMetaValue(key, pairs[i].Value, getTextTitle, evalMeta, enc)
			metadata = metadata.Cons(sx.Cons(sx.MakeString(key), sxval))
		}

		summary := collect.References(zn)
		locLinks, queryLinks, extLinks := wui.splitLocSeaExtLinks(append(summary.Links, summary.Embeds...))

		title := parser.NormalizedSpacedText(zn.InhMeta.GetTitle())
		phrase := q.Get(api.QueryKeyPhrase)
		if phrase == "" {
			phrase = title
		}
		phrase = strings.TrimSpace(phrase)
		unlinkedMeta, err := unlinkedRefs.Run(ctx, phrase, adapter.AddUnlinkedRefsToQuery(query.Parse("ORDER id"), zn.InhMeta))
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		bns := evaluate.RunBlockNode(ctx, evaluator.QueryAction(ctx, nil, unlinkedMeta, wui.rtConfig))
		unlinkedContent, _, err := enc.BlocksSxn(&bns)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		encTexts := encodingTexts()
		shadowLinks := getShadowLinks(ctx, zid, getAllMeta)

		user := server.GetUser(ctx)
		env, rb := wui.createRenderEnv(ctx, "info", wui.rtConfig.Get(ctx, nil, api.KeyLang), title, user)
		rb.bindString("metadata", metadata)
		rb.bindString("local-links", locLinks)
		rb.bindString("query-links", queryLinks)
		rb.bindString("ext-links", extLinks)
		rb.bindString("unlinked-content", unlinkedContent)
		rb.bindString("phrase", sx.MakeString(phrase))
		rb.bindString("query-key-phrase", sx.MakeString(api.QueryKeyPhrase))
		rb.bindString("enc-eval", wui.infoAPIMatrix(zid, false, encTexts))
		rb.bindString("enc-parsed", wui.infoAPIMatrixParsed(zid, encTexts))
		rb.bindString("shadow-links", shadowLinks)
		wui.bindCommonZettelData(ctx, &rb, user, zn.InhMeta, &zn.Content)
		if rb.err == nil {
			err = wui.renderSxnTemplate(ctx, w, id.InfoTemplateZid, env)
		}
		if err != nil {
			wui.reportError(ctx, w, err)
		}
	}
}

func (wui *WebUI) splitLocSeaExtLinks(links []*ast.Reference) (locLinks, queries, extLinks *sx.Pair) {
	for i := len(links) - 1; i >= 0; i-- {
		ref := links[i]
		if ref.State == ast.RefStateSelf || ref.IsZettel() {
			continue
		}
		if ref.State == ast.RefStateQuery {
			queries = queries.Cons(
				sx.Cons(
					sx.MakeString(ref.Value),
					sx.MakeString(wui.NewURLBuilder('h').AppendQuery(ref.Value).String())))
			continue
		}
		if ref.IsExternal() {
			extLinks = extLinks.Cons(sx.MakeString(ref.String()))
			continue
		}
		locLinks = locLinks.Cons(sx.Cons(sx.MakeBoolean(ref.IsValid()), sx.MakeString(ref.String())))
	}
	return locLinks, queries, extLinks
}

func encodingTexts() []string {
	encodings := encoder.GetEncodings()
	encTexts := make([]string, 0, len(encodings))
	for _, f := range encodings {
		encTexts = append(encTexts, f.String())
	}
	sort.Strings(encTexts)
	return encTexts
}

var apiParts = []string{api.PartZettel, api.PartMeta, api.PartContent}

func (wui *WebUI) infoAPIMatrix(zid id.Zid, parseOnly bool, encTexts []string) *sx.Pair {
	matrix := sx.Nil()
	u := wui.NewURLBuilder('z').SetZid(api.ZettelID(zid.String()))
	for ip := len(apiParts) - 1; ip >= 0; ip-- {
		part := apiParts[ip]
		row := sx.Nil()
		for je := len(encTexts) - 1; je >= 0; je-- {
			enc := encTexts[je]
			if parseOnly {
				u.AppendKVQuery(api.QueryKeyParseOnly, "")
			}
			u.AppendKVQuery(api.QueryKeyPart, part)
			u.AppendKVQuery(api.QueryKeyEncoding, enc)
			row = row.Cons(sx.Cons(sx.MakeString(enc), sx.MakeString(u.String())))
			u.ClearQuery()
		}
		matrix = matrix.Cons(sx.Cons(sx.MakeString(part), row))
	}
	return matrix
}

func (wui *WebUI) infoAPIMatrixParsed(zid id.Zid, encTexts []string) *sx.Pair {
	matrix := wui.infoAPIMatrix(zid, true, encTexts)
	u := wui.NewURLBuilder('z').SetZid(api.ZettelID(zid.String()))

	for i, row := 0, matrix; i < len(apiParts) && row != nil; row = row.Tail() {
		line, isLine := sx.GetPair(row.Car())
		if !isLine || line == nil {
			continue
		}
		last := line.LastPair()
		part := apiParts[i]
		u.AppendKVQuery(api.QueryKeyPart, part)
		last = last.AppendBang(sx.Cons(sx.MakeString("plain"), sx.MakeString(u.String())))
		u.ClearQuery()
		if i < 2 {
			u.AppendKVQuery(api.QueryKeyEncoding, api.EncodingData)
			u.AppendKVQuery(api.QueryKeyPart, part)
			last = last.AppendBang(sx.Cons(sx.MakeString("data"), sx.MakeString(u.String())))
			u.ClearQuery()
			u.AppendKVQuery(api.QueryKeyEncoding, api.EncodingJson)
			u.AppendKVQuery(api.QueryKeyPart, part)
			last.AppendBang(sx.Cons(sx.MakeString("json"), sx.MakeString(u.String())))
			u.ClearQuery()
		}
		i++
	}
	return matrix
}

func getShadowLinks(ctx context.Context, zid id.Zid, getAllZettel usecase.GetAllZettel) *sx.Pair {
	result := sx.Nil()
	if zl, err := getAllZettel.Run(ctx, zid); err == nil {
		for i := len(zl) - 1; i >= 1; i-- {
			if boxNo, ok := zl[i].Meta.Get(api.KeyBoxNumber); ok {
				result = result.Cons(sx.MakeString(boxNo))
			}
		}
	}
	return result
}
