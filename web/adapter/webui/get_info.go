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
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"zettelstore.de/z/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/collect"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
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
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		q := r.URL.Query()
		if enc, encText := adapter.GetEncoding(r, q, api.EncoderHTML); enc != api.EncoderHTML {
			wui.reportError(ctx, w, adapter.NewErrBadRequest(
				fmt.Sprintf("Zettel info not available in encoding %q", encText)))
			return
		}

		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			wui.reportError(ctx, w, box.ErrNotFound)
			return
		}

		zn, err := parseZettel.Run(ctx, zid, q.Get(meta.KeySyntax))
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}

		summary := collect.References(zn)
		locLinks, extLinks := splitLocExtLinks(append(summary.Links, summary.Embeds...))

		envEval := evaluator.Environment{
			EmbedImage: true,
			GetTagRef: func(s string) *ast.Reference {
				return adapter.CreateTagReference(wui, 'h', api.EncodingHTML, s)
			},
			GetHostedRef: func(s string) *ast.Reference {
				return adapter.CreateHostedReference(wui, s)
			},
			GetFoundRef: func(zid id.Zid, fragment string) *ast.Reference {
				return adapter.CreateFoundReference(wui, 'h', "", "", zid, fragment)
			},
		}
		lang := config.GetLang(zn.InhMeta, wui.rtConfig)
		envHTML := encoder.Environment{Lang: lang}
		pairs := zn.Meta.Pairs(true)
		metaData := make([]metaDataInfo, len(pairs))
		getTextTitle := wui.makeGetTextTitle(ctx, getMeta, evaluate)
		for i, p := range pairs {
			var html strings.Builder
			wui.writeHTMLMetaValue(ctx, &html, p.Key, p.Value, getTextTitle, evaluate, &envEval, &envHTML)
			metaData[i] = metaDataInfo{p.Key, html.String()}
		}
		shadowLinks := getShadowLinks(ctx, zid, getAllMeta)
		endnotes, err := encodeBlocks(&ast.BlockListNode{}, api.EncoderHTML, &envHTML)
		if err != nil {
			endnotes = ""
		}

		textTitle := wui.encodeTitleAsText(ctx, zn.InhMeta, evaluate)
		user := wui.getUser(ctx)
		canCreate := wui.canCreate(ctx, user)
		var base baseData
		wui.makeBaseData(ctx, lang, textTitle, user, &base)
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
			HasLinks       bool
			HasLocLinks    bool
			LocLinks       []localLink
			HasExtLinks    bool
			ExtLinks       []string
			ExtNewWindow   string
			EvalMatrix     []matrixLine
			ParseMatrix    []matrixLine
			HasShadowLinks bool
			ShadowLinks    []string
			Endnotes       string
		}{
			Zid:            zid.String(),
			WebURL:         wui.NewURLBuilder('h').SetZid(zid).String(),
			ContextURL:     wui.NewURLBuilder('k').SetZid(zid).String(),
			CanWrite:       wui.canWrite(ctx, user, zn.Meta, zn.Content),
			EditURL:        wui.NewURLBuilder('e').SetZid(zid).String(),
			CanFolge:       canCreate,
			FolgeURL:       wui.NewURLBuilder('f').SetZid(zid).String(),
			CanCopy:        canCreate && !zn.Content.IsBinary(),
			CopyURL:        wui.NewURLBuilder('c').SetZid(zid).String(),
			CanRename:      wui.canRename(ctx, user, zn.Meta),
			RenameURL:      wui.NewURLBuilder('b').SetZid(zid).String(),
			CanDelete:      wui.canDelete(ctx, user, zn.Meta),
			DeleteURL:      wui.NewURLBuilder('d').SetZid(zid).String(),
			MetaData:       metaData,
			HasLinks:       len(extLinks)+len(locLinks) > 0,
			HasLocLinks:    len(locLinks) > 0,
			LocLinks:       locLinks,
			HasExtLinks:    len(extLinks) > 0,
			ExtLinks:       extLinks,
			ExtNewWindow:   htmlAttrNewWindow(len(extLinks) > 0),
			EvalMatrix:     wui.infoAPIMatrix('v', zid),
			ParseMatrix:    wui.infoAPIMatrixPlain('u', zid),
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
	parts := []string{api.PartZettel, api.PartMeta, api.PartContent}
	matrix := make([]matrixLine, 0, len(parts))
	u := wui.NewURLBuilder(key).SetZid(zid)
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

	// Append plain and JSON format
	u := wui.NewURLBuilder('p').SetZid(zid)
	parts := []string{api.PartZettel, api.PartMeta, api.PartContent}
	for i, part := range parts {
		u.AppendQuery(api.QueryKeyPart, part)
		matrix[i].Elements = append(matrix[i].Elements, simpleLink{"plain", u.String()})
		u.ClearQuery()
	}
	u = wui.NewURLBuilder('j').SetZid(zid)
	matrix[0].Elements = append(matrix[0].Elements, simpleLink{"json", u.String()})
	return matrix
}

func getShadowLinks(ctx context.Context, zid id.Zid, getAllMeta usecase.GetAllMeta) []string {
	ml, err := getAllMeta.Run(ctx, zid)
	if err != nil || len(ml) < 2 {
		return nil
	}
	result := make([]string, 0, len(ml)-1)
	for _, m := range ml[1:] {
		if boxNo, ok := m.Get(meta.KeyBoxNumber); ok {
			result = append(result, boxNo)
		}
	}
	return result
}
