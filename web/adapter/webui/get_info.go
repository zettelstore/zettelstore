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
	"context"
	"net/http"
	"sort"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/collect"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/evaluator"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

type metaDataInfo struct {
	Key   string
	Value string
}

type matrixLine struct {
	Header   string
	Elements []simpleLink
}

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

		envEval := evaluator.Environment{
			GetTagRef:        wui.createTagReference,
			GetHostedRef:     wui.createHostedReference,
			GetFoundRef:      wui.createFoundReference,
			GetImageMaterial: wui.createImageMaterial,
		}
		enc := wui.getSimpleHTMLEncoder()

		pairs := zn.Meta.ComputedPairs()
		metaData := make([]metaDataInfo, len(pairs))
		getTextTitle := wui.makeGetTextTitle(ctx, getMeta, evaluate)
		for i, p := range pairs {
			var buf bytes.Buffer
			wui.writeHTMLMetaValue(
				&buf, p.Key, p.Value,
				getTextTitle,
				func(val string) ast.InlineSlice {
					return evaluate.RunMetadata(ctx, val, &envEval)
				},
				enc)
			metaData[i] = metaDataInfo{p.Key, buf.String()}
		}
		summary := collect.References(zn)
		locLinks, extLinks := splitLocExtLinks(append(summary.Links, summary.Embeds...))

		textTitle := wui.encodeTitleAsText(ctx, zn.InhMeta, evaluate)
		phrase := q.Get(api.QueryKeyPhrase)
		if phrase == "" {
			phrase = textTitle
		}
		phrase = strings.TrimSpace(phrase)
		unlinkedMeta, err := unlinkedRefs.Run(
			ctx, phrase, adapter.AddUnlinkedRefsToSearch(nil, zn.InhMeta))
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		unLinks := wui.buildHTMLMetaList(ctx, unlinkedMeta, evaluate)

		shadowLinks := getShadowLinks(ctx, zid, getAllMeta)
		endnotes, err := enc.BlocksString(&ast.BlockSlice{})
		if err != nil {
			endnotes = ""
		}

		user := wui.getUser(ctx)
		canCreate := wui.canCreate(ctx, user)
		apiZid := api.ZettelID(zid.String())
		var base baseData
		wui.makeBaseData(ctx, config.GetLang(zn.InhMeta, wui.rtConfig), textTitle, "", user, &base)
		wui.renderTemplate(ctx, w, id.InfoTemplateZid, &base, struct {
			Zid            string
			WebURL         string
			ContextURL     string
			CanWrite       bool
			EditURL        string
			CanFolge       bool
			FolgeURL       string
			CanCopy        bool
			CopyURL        string
			CanRename      bool
			RenameURL      string
			CanDelete      bool
			DeleteURL      string
			MetaData       []metaDataInfo
			HasLocLinks    bool
			LocLinks       []localLink
			HasExtLinks    bool
			ExtLinks       []string
			ExtNewWindow   string
			UnLinks        []simpleLink
			UnLinksPhrase  string
			QueryKeyPhrase string
			EvalMatrix     []matrixLine
			ParseMatrix    []matrixLine
			HasShadowLinks bool
			ShadowLinks    []string
			Endnotes       string
		}{
			Zid:            zid.String(),
			WebURL:         wui.NewURLBuilder('h').SetZid(apiZid).String(),
			ContextURL:     wui.NewURLBuilder('k').SetZid(apiZid).String(),
			CanWrite:       wui.canWrite(ctx, user, zn.Meta, zn.Content),
			EditURL:        wui.NewURLBuilder('e').SetZid(apiZid).String(),
			CanFolge:       canCreate,
			FolgeURL:       wui.NewURLBuilder('c').SetZid(apiZid).AppendQuery(queryKeyAction, valueActionFolge).String(),
			CanCopy:        canCreate && !zn.Content.IsBinary(),
			CopyURL:        wui.NewURLBuilder('c').SetZid(apiZid).AppendQuery(queryKeyAction, valueActionCopy).String(),
			CanRename:      wui.canRename(ctx, user, zn.Meta),
			RenameURL:      wui.NewURLBuilder('b').SetZid(apiZid).String(),
			CanDelete:      wui.canDelete(ctx, user, zn.Meta),
			DeleteURL:      wui.NewURLBuilder('d').SetZid(apiZid).String(),
			MetaData:       metaData,
			HasLocLinks:    len(locLinks) > 0,
			LocLinks:       locLinks,
			HasExtLinks:    len(extLinks) > 0,
			ExtLinks:       extLinks,
			ExtNewWindow:   htmlAttrNewWindow(len(extLinks) > 0),
			UnLinks:        unLinks,
			UnLinksPhrase:  phrase,
			QueryKeyPhrase: api.QueryKeyPhrase,
			EvalMatrix:     wui.infoAPIMatrix('v', zid),
			ParseMatrix:    wui.infoAPIMatrixPlain('p', zid),
			HasShadowLinks: len(shadowLinks) > 0,
			ShadowLinks:    shadowLinks,
			Endnotes:       endnotes,
		})
	}
}

type localLink struct {
	Valid bool
	Zid   string
}

func splitLocExtLinks(links []*ast.Reference) (locLinks []localLink, extLinks []string) {
	if len(links) == 0 {
		return nil, nil
	}
	for _, ref := range links {
		if ref.State == ast.RefStateSelf {
			continue
		}
		if ref.IsZettel() {
			continue
		}
		if ref.IsExternal() {
			extLinks = append(extLinks, ref.String())
			continue
		}
		locLinks = append(locLinks, localLink{ref.IsValid(), ref.String()})
	}
	return locLinks, extLinks
}

func (wui *WebUI) infoAPIMatrix(key byte, zid id.Zid) []matrixLine {
	encodings := encoder.GetEncodings()
	encTexts := make([]string, 0, len(encodings))
	for _, f := range encodings {
		encTexts = append(encTexts, f.String())
	}
	sort.Strings(encTexts)
	defEncoding := encoder.GetDefaultEncoding().String()
	parts := getParts()
	matrix := make([]matrixLine, 0, len(parts))
	u := wui.NewURLBuilder(key).SetZid(api.ZettelID(zid.String()))
	for _, part := range parts {
		row := make([]simpleLink, len(encTexts))
		for j, enc := range encTexts {
			u.AppendQuery(api.QueryKeyPart, part)
			if enc != defEncoding {
				u.AppendQuery(api.QueryKeyEncoding, enc)
			}
			row[j] = simpleLink{enc, u.String()}
			u.ClearQuery()
		}
		matrix = append(matrix, matrixLine{part, row})
	}
	return matrix
}

func (wui *WebUI) infoAPIMatrixPlain(key byte, zid id.Zid) []matrixLine {
	matrix := wui.infoAPIMatrix(key, zid)
	apiZid := api.ZettelID(zid.String())

	// Append plain and JSON format
	u := wui.NewURLBuilder('z').SetZid(apiZid)
	for i, part := range getParts() {
		u.AppendQuery(api.QueryKeyPart, part)
		matrix[i].Elements = append(matrix[i].Elements, simpleLink{"plain", u.String()})
		u.ClearQuery()
	}
	u = wui.NewURLBuilder('j').SetZid(apiZid)
	matrix[0].Elements = append(matrix[0].Elements, simpleLink{"json", u.String()})
	u = wui.NewURLBuilder('m').SetZid(apiZid)
	matrix[1].Elements = append(matrix[1].Elements, simpleLink{"json", u.String()})
	return matrix
}

func getParts() []string {
	return []string{api.PartZettel, api.PartMeta, api.PartContent}
}

func getShadowLinks(ctx context.Context, zid id.Zid, getAllMeta usecase.GetAllMeta) []string {
	ml, err := getAllMeta.Run(ctx, zid)
	if err != nil || len(ml) < 2 {
		return nil
	}
	result := make([]string, 0, len(ml)-1)
	for _, m := range ml[1:] {
		if boxNo, ok := m.Get(api.KeyBoxNumber); ok {
			result = append(result, boxNo)
		}
	}
	return result
}
