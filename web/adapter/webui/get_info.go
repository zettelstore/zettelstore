//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package webui provides wet-UI handlers for web requests.
package webui

import (
	"fmt"
	"net/http"
	"strings"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/collect"
	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/session"
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
		q := r.URL.Query()
		if format := adapter.GetFormat(r, q, "html"); format != "html" {
			adapter.BadRequest(w, fmt.Sprintf("Zettel info not available in format %q", format))
			return
		}

		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		zn, err := parseZettel.Run(ctx, zid, q.Get("syntax"))
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}

		langOption := &encoder.StringOption{
			Key:   "lang",
			Value: runtime.GetLang(zn.InhMeta)}

		summary := collect.References(zn)
		locLinks, extLinks := splitLocExtLinks(append(summary.Links, summary.Images...))

		textTitle, err := adapter.FormatInlines(zn.Title, "text", nil, langOption)
		if err != nil {
			adapter.InternalServerError(w, "Format Text inlines for info", err)
			return
		}

		pairs := zn.Zettel.Meta.Pairs(true)
		metaData := make([]metaDataInfo, 0, len(pairs))
		getTitle := makeGetTitle(ctx, getMeta, langOption)
		for _, p := range pairs {
			var html strings.Builder
			writeHTMLMetaValue(&html, zn.Zettel.Meta, p.Key, getTitle, langOption)
			metaData = append(metaData, metaDataInfo{p.Key, html.String()})
		}

		user := session.GetUser(ctx)
		var base baseData
		te.makeBaseData(ctx, langOption.Value, textTitle, user, &base)
		canCopy := base.CanCreate && !zn.Zettel.Content.IsBinary()
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
			LocLinks     []string
			HasExtLinks  bool
			ExtLinks     []string
			ExtNewWindow string
			Matrix       []matrixLine
		}{
			Zid:          zid.String(),
			WebURL:       adapter.NewURLBuilder('h').SetZid(zid).String(),
			ContextURL:   adapter.NewURLBuilder('j').SetZid(zid).String(),
			CanWrite:     te.canWrite(ctx, user, zn.Zettel),
			EditURL:      adapter.NewURLBuilder('e').SetZid(zid).String(),
			CanFolge:     base.CanCreate && !zn.Zettel.Content.IsBinary(),
			FolgeURL:     adapter.NewURLBuilder('f').SetZid(zid).String(),
			CanCopy:      canCopy,
			CopyURL:      adapter.NewURLBuilder('c').SetZid(zid).String(),
			CanRename:    te.canRename(ctx, user, zn.Zettel.Meta),
			RenameURL:    adapter.NewURLBuilder('b').SetZid(zid).String(),
			CanDelete:    te.canDelete(ctx, user, zn.Zettel.Meta),
			DeleteURL:    adapter.NewURLBuilder('d').SetZid(zid).String(),
			MetaData:     metaData,
			HasLinks:     len(extLinks)+len(locLinks) > 0,
			HasLocLinks:  len(locLinks) > 0,
			LocLinks:     locLinks,
			HasExtLinks:  len(extLinks) > 0,
			ExtLinks:     extLinks,
			ExtNewWindow: htmlAttrNewWindow(len(extLinks) > 0),
			Matrix:       infoAPIMatrix(zid),
		})
	}
}

func splitLocExtLinks(links []*ast.Reference) (locLinks, extLinks []string) {
	if len(links) == 0 {
		return nil, nil
	}
	for _, ref := range links {
		if ref.State == ast.RefStateSelf {
			continue
		}
		if ref.IsZettel() {
			continue
		} else if ref.IsExternal() {
			extLinks = append(extLinks, ref.String())
		} else {
			locLinks = append(locLinks, ref.String())
		}
	}
	return locLinks, extLinks
}

func infoAPIMatrix(zid id.Zid) []matrixLine {
	formats := encoder.GetFormats()
	defFormat := encoder.GetDefaultFormat()
	parts := []string{"zettel", "meta", "content"}
	matrix := make([]matrixLine, 0, len(parts))
	u := adapter.NewURLBuilder('z').SetZid(zid)
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
