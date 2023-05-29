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

	"codeberg.org/t73fde/sxpf"
	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/collect"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/evaluator"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/server"
	"zettelstore.de/z/zettel/id"
)

// MakeGetInfoHandler creates a new HTTP handler for the use case "get zettel".
func (wui *WebUI) MakeGetInfoHandler(
	parseZettel usecase.ParseZettel,
	evaluate *usecase.Evaluate,
	getMeta usecase.GetMeta,
	getAllMeta usecase.GetAllMeta,
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
		getTextTitle := wui.makeGetTextTitle(ctx, getMeta)
		evalMeta := func(val string) ast.InlineSlice {
			return evaluate.RunMetadata(ctx, val)
		}
		pairs := zn.Meta.ComputedPairs()
		metadata := sxpf.Nil()
		for i := len(pairs) - 1; i >= 0; i-- {
			key := pairs[i].Key
			sxval := wui.writeHTMLMetaValue(key, pairs[i].Value, getTextTitle, evalMeta, enc)
			metadata = metadata.Cons(sxpf.Cons(sxpf.MakeString(key), sxval))
		}

		summary := collect.References(zn)
		locLinks, queryLinks, extLinks := wui.splitLocSeaExtLinks(append(summary.Links, summary.Embeds...))

		title := parser.NormalizedSpacedText(zn.InhMeta.GetTitle())
		phrase := q.Get(api.QueryKeyPhrase)
		if phrase == "" {
			phrase = title
		}
		phrase = strings.TrimSpace(phrase)
		unlinkedMeta, err := unlinkedRefs.Run(ctx, phrase, adapter.AddUnlinkedRefsToQuery(nil, zn.InhMeta))
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
		canCreate := wui.canCreate(ctx, user)
		apiZid := api.ZettelID(zid.String())
		env, err := wui.createRenderEnv(ctx, "info", wui.rtConfig.Get(ctx, nil, api.KeyLang), title, user)
		rb := makeRenderBinder(wui.sf, env, err)
		rb.bindString("zid", sxpf.MakeString(zid.String()))
		rb.bindString("web-url", sxpf.MakeString(wui.NewURLBuilder('h').SetZid(apiZid).String()))
		rb.bindString("context-url", sxpf.MakeString(wui.NewURLBuilder('h').AppendQuery(api.ContextDirective+" "+zid.String()).String()))
		if wui.canWrite(ctx, user, zn.Meta, zn.Content) {
			rb.bindString("edit-url", sxpf.MakeString(wui.NewURLBuilder('e').SetZid(apiZid).String()))
		}
		if canCreate && !zn.Content.IsBinary() {
			rb.bindString("copy-url", sxpf.MakeString(wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionCopy).String()))
		}
		if canCreate {
			rb.bindString("version-url", sxpf.MakeString(wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionVersion).String()))
			rb.bindString("folge-url", sxpf.MakeString(wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionFolge).String()))
		}
		if wui.canRename(ctx, user, zn.Meta) {
			rb.bindString("rename-url", sxpf.MakeString(wui.NewURLBuilder('b').SetZid(apiZid).String()))
		}
		if wui.canDelete(ctx, user, zn.Meta) {
			rb.bindString("delete-url", sxpf.MakeString(wui.NewURLBuilder('d').SetZid(apiZid).String()))
		}
		rb.bindString("metadata", metadata)
		rb.bindString("local-links", locLinks)
		rb.bindString("query-links", queryLinks)
		rb.bindString("ext-links", extLinks)
		rb.bindString("unlinked-content", unlinkedContent)
		rb.bindString("phrase", sxpf.MakeString(phrase))
		rb.bindString("query-key-phrase", sxpf.MakeString(api.QueryKeyPhrase))
		rb.bindString("enc-eval", wui.infoAPIMatrix(zid, false, encTexts))
		rb.bindString("enc-parsed", wui.infoAPIMatrixParsed(zid, encTexts))
		rb.bindString("shadow-links", shadowLinks)
		if rb.err == nil {
			err = wui.renderSxnTemplate(ctx, w, id.InfoTemplateZid, env)
		}
		if err != nil {
			wui.reportError(ctx, w, err)
		}
	}
}

func (wui *WebUI) splitLocSeaExtLinks(links []*ast.Reference) (locLinks, queries, extLinks *sxpf.List) {
	for i := len(links) - 1; i >= 0; i-- {
		ref := links[i]
		if ref.State == ast.RefStateSelf || ref.IsZettel() {
			continue
		}
		if ref.State == ast.RefStateQuery {
			queries = queries.Cons(
				sxpf.Cons(
					sxpf.MakeString(ref.Value),
					sxpf.MakeString(wui.NewURLBuilder('h').AppendQuery(ref.Value).String())))
			continue
		}
		if ref.IsExternal() {
			extLinks = extLinks.Cons(sxpf.MakeString(ref.String()))
			continue
		}
		locLinks = locLinks.Cons(sxpf.Cons(sxpf.MakeBoolean(ref.IsValid()), sxpf.MakeString(ref.String())))
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

func (wui *WebUI) infoAPIMatrix(zid id.Zid, parseOnly bool, encTexts []string) *sxpf.List {
	matrix := sxpf.Nil()
	u := wui.NewURLBuilder('z').SetZid(api.ZettelID(zid.String()))
	for ip := len(apiParts) - 1; ip >= 0; ip-- {
		part := apiParts[ip]
		row := sxpf.Nil()
		for je := len(encTexts) - 1; je >= 0; je-- {
			enc := encTexts[je]
			if parseOnly {
				u.AppendKVQuery(api.QueryKeyParseOnly, "")
			}
			u.AppendKVQuery(api.QueryKeyPart, part)
			u.AppendKVQuery(api.QueryKeyEncoding, enc)
			row = row.Cons(sxpf.Cons(sxpf.MakeString(enc), sxpf.MakeString(u.String())))
			u.ClearQuery()
		}
		matrix = matrix.Cons(sxpf.Cons(sxpf.MakeString(part), row))
	}
	return matrix
}

func (wui *WebUI) infoAPIMatrixParsed(zid id.Zid, encTexts []string) *sxpf.List {
	matrix := wui.infoAPIMatrix(zid, true, encTexts)
	// apiZid := api.ZettelID(zid.String())
	u := wui.NewURLBuilder('z').SetZid(api.ZettelID(zid.String()))

	for i, row := 0, matrix; i < len(apiParts) && row != nil; row = row.Tail() {
		line, isLine := sxpf.GetList(row.Car())
		if !isLine || line == nil {
			continue
		}
		last := line.LastPair()
		part := apiParts[i]
		u.AppendKVQuery(api.QueryKeyPart, part)
		last = last.AppendBang(sxpf.Cons(sxpf.MakeString("plain"), sxpf.MakeString(u.String())))
		u.ClearQuery()
		if i < 2 {
			u.AppendKVQuery(api.QueryKeyEncoding, api.EncodingData)
			u.AppendKVQuery(api.QueryKeyPart, part)
			last = last.AppendBang(sxpf.Cons(sxpf.MakeString("data"), sxpf.MakeString(u.String())))
			u.ClearQuery()
			u.AppendKVQuery(api.QueryKeyEncoding, api.EncodingJson)
			u.AppendKVQuery(api.QueryKeyPart, part)
			last.AppendBang(sxpf.Cons(sxpf.MakeString("json"), sxpf.MakeString(u.String())))
			u.ClearQuery()
		}
		i++
	}
	return matrix
}

func getShadowLinks(ctx context.Context, zid id.Zid, getAllMeta usecase.GetAllMeta) *sxpf.List {
	result := sxpf.Nil()
	if ml, err := getAllMeta.Run(ctx, zid); err == nil {
		for i := len(ml) - 1; i >= 1; i-- {
			if boxNo, ok := ml[i].Get(api.KeyBoxNumber); ok {
				result = result.Cons(sxpf.MakeString(boxNo))
			}
		}
	}
	return result
}
