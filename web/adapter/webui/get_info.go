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
	"fmt"
	"net/http"
	"strings"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/collect"
	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/encoder/encfun"
	"zettelstore.de/z/place"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/server"
)

type metaDataInfo struct {
	Key   string
	Value string
}

type matrixElement struct {
	Text   string
	HasURL bool
	URL    string
}
type matrixLine struct {
	Elements []matrixElement
}

// MakeGetInfoHandler creates a new HTTP handler for the use case "get zettel".
func MakeGetInfoHandler(
	te *TemplateEngine,
	parseZettel usecase.ParseZettel,
	getMeta usecase.GetMeta,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		q := r.URL.Query()
		if format := adapter.GetFormat(r, q, "html"); format != "html" {
			te.reportError(ctx, w, adapter.NewErrBadRequest(
				fmt.Sprintf("Zettel info not available in format %q", format)))
			return
		}

		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			te.reportError(ctx, w, place.ErrNotFound)
			return
		}

		zn, err := parseZettel.Run(ctx, zid, q.Get("syntax"))
		if err != nil {
			te.reportError(ctx, w, err)
			return
		}

		summary := collect.References(zn)
		locLinks, extLinks := splitLocExtLinks(append(summary.Links, summary.Images...))

		lang := runtime.GetLang(zn.InhMeta)
		env := encoder.Environment{Lang: lang}
		pairs := zn.Meta.Pairs(true)
		metaData := make([]metaDataInfo, len(pairs))
		getTitle := makeGetTitle(ctx, getMeta, &env)
		builder := server.GetURLBuilderFunc(ctx)
		for i, p := range pairs {
			var html strings.Builder
			writeHTMLMetaValue(&html, builder, zn.Meta, p.Key, getTitle, &env)
			metaData[i] = metaDataInfo{p.Key, html.String()}
		}
		endnotes, err := formatBlocks(nil, "html", &env)
		if err != nil {
			endnotes = ""
		}

		textTitle := encfun.MetaAsText(zn.InhMeta, meta.KeyTitle)
		user := server.GetUser(ctx)
		var base baseData
		te.makeBaseData(ctx, lang, textTitle, user, &base)
		canCopy := base.CanCreate && !zn.Content.IsBinary()
		te.renderTemplate(ctx, w, id.InfoTemplateZid, &base, struct {
			Zid          string
			WebURL       string
			ContextURL   string
			CanWrite     bool
			EditURL      string
			CanFolge     bool
			FolgeURL     string
			CanCopy      bool
			CopyURL      string
			CanRename    bool
			RenameURL    string
			CanDelete    bool
			DeleteURL    string
			MetaData     []metaDataInfo
			HasLinks     bool
			HasLocLinks  bool
			LocLinks     []localLink
			HasExtLinks  bool
			ExtLinks     []string
			ExtNewWindow string
			Matrix       []matrixLine
			Endnotes     string
		}{
			Zid:          zid.String(),
			WebURL:       builder('h').SetZid(zid).String(),
			ContextURL:   builder('j').SetZid(zid).String(),
			CanWrite:     te.canWrite(ctx, user, zn.Meta, zn.Content),
			EditURL:      builder('e').SetZid(zid).String(),
			CanFolge:     base.CanCreate && !zn.Content.IsBinary(),
			FolgeURL:     builder('f').SetZid(zid).String(),
			CanCopy:      canCopy,
			CopyURL:      builder('c').SetZid(zid).String(),
			CanRename:    te.canRename(ctx, user, zn.Meta),
			RenameURL:    builder('b').SetZid(zid).String(),
			CanDelete:    te.canDelete(ctx, user, zn.Meta),
			DeleteURL:    builder('d').SetZid(zid).String(),
			MetaData:     metaData,
			HasLinks:     len(extLinks)+len(locLinks) > 0,
			HasLocLinks:  len(locLinks) > 0,
			LocLinks:     locLinks,
			HasExtLinks:  len(extLinks) > 0,
			ExtLinks:     extLinks,
			ExtNewWindow: htmlAttrNewWindow(len(extLinks) > 0),
			Matrix:       infoAPIMatrix(builder, zid),
			Endnotes:     endnotes,
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

func infoAPIMatrix(builder server.URLBuilderFunc, zid id.Zid) []matrixLine {
	formats := encoder.GetFormats()
	defFormat := encoder.GetDefaultFormat()
	parts := []string{"zettel", "meta", "content"}
	matrix := make([]matrixLine, 0, len(parts))
	u := builder('z').SetZid(zid)
	for _, part := range parts {
		row := make([]matrixElement, 0, len(formats)+1)
		row = append(row, matrixElement{part, false, ""})
		for _, format := range formats {
			u.AppendQuery("_part", part)
			if format != defFormat {
				u.AppendQuery("_format", format)
			}
			row = append(row, matrixElement{format, true, u.String()})
			u.ClearQuery()
		}
		matrix = append(matrix, matrixLine{row})
	}
	return matrix
}
