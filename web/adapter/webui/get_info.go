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
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/place"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/session"
)

type metaDataInfo struct {
	Key   string
	Value string
}

type zettelReference struct {
	Zid    id.Zid
	Title  string
	HasURL bool
	URL    string
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
		getTitle := func(zid id.Zid, format string) (string, int) {
			m, err := getMeta.Run(r.Context(), zid)
			if err != nil {
				if place.IsErrNotAllowed(err) {
					return "", -1
				}
				return "", 0
			}
			astTitle := parser.ParseTitle(m.GetDefault(meta.KeyTitle, ""))
			title, err := adapter.FormatInlines(astTitle, format, langOption)
			if err == nil {
				return title, 1
			}
			return "", 1
		}
		summary := collect.References(zn)
		zetLinks, locLinks, extLinks := splitIntExtLinks(
			getTitle, append(summary.Links, summary.Images...))

		textTitle, err := adapter.FormatInlines(zn.Title, "text", nil, langOption)
		if err != nil {
			adapter.InternalServerError(w, "Format Text inlines for info", err)
			return
		}

		user := session.GetUser(ctx)
		pairs := zn.Zettel.Meta.Pairs(true)
		metaData := make([]metaDataInfo, 0, len(pairs))
		for _, p := range pairs {
			var html strings.Builder
			writeHTMLMetaValue(&html, zn.Zettel.Meta, p.Key, getTitle, langOption)
			metaData = append(metaData, metaDataInfo{p.Key, html.String()})
		}
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
		var base baseData
		te.makeBaseData(ctx, langOption.Value, textTitle, user, &base)
		canCopy := base.CanCreate && !zn.Zettel.Content.IsBinary()
		te.renderTemplate(ctx, w, id.InfoTemplateZid, &base, struct {
			Zid          string
			WebURL       string
			CanWrite     bool
			EditURL      string
			CanFolge     bool
			FolgeURL     string
			CanCopy      bool
			CopyURL      string
			CanNew       bool
			NewURL       string
			CanRename    bool
			RenameURL    string
			CanDelete    bool
			DeleteURL    string
			MetaData     []metaDataInfo
			HasLinks     bool
			HasZetLinks  bool
			ZetLinks     []zettelReference
			HasLocLinks  bool
			LocLinks     []string
			HasExtLinks  bool
			ExtLinks     []string
			ExtNewWindow string
			Matrix       []matrixLine
		}{
			Zid:      zid.String(),
			WebURL:   adapter.NewURLBuilder('h').SetZid(zid).String(),
			CanWrite: te.canWrite(ctx, user, zn.Zettel),
			EditURL:  adapter.NewURLBuilder('e').SetZid(zid).String(),
			CanFolge: base.CanCreate && !zn.Zettel.Content.IsBinary(),
			FolgeURL: adapter.NewURLBuilder('f').SetZid(zid).String(),
			CanCopy:  canCopy,
			CopyURL:  adapter.NewURLBuilder('c').SetZid(zid).String(),
			CanNew: canCopy && zn.Zettel.Meta.GetDefault(meta.KeyRole, "") ==
				meta.ValueRoleNewTemplate,
			NewURL:       adapter.NewURLBuilder('n').SetZid(zid).String(),
			CanRename:    te.canRename(ctx, user, zn.Zettel.Meta),
			RenameURL:    adapter.NewURLBuilder('r').SetZid(zid).String(),
			CanDelete:    te.canDelete(ctx, user, zn.Zettel.Meta),
			DeleteURL:    adapter.NewURLBuilder('d').SetZid(zid).String(),
			MetaData:     metaData,
			HasLinks:     len(zetLinks)+len(extLinks)+len(locLinks) > 0,
			HasZetLinks:  len(zetLinks) > 0,
			ZetLinks:     zetLinks,
			HasLocLinks:  len(locLinks) > 0,
			LocLinks:     locLinks,
			HasExtLinks:  len(extLinks) > 0,
			ExtLinks:     extLinks,
			ExtNewWindow: htmlAttrNewWindow(len(extLinks) > 0),
			Matrix:       matrix,
		})
	}
}

func splitIntExtLinks(
	getTitle func(id.Zid, string) (string, int),
	links []*ast.Reference,
) (zetLinks []zettelReference, locLinks []string, extLinks []string) {
	if len(links) == 0 {
		return nil, nil, nil
	}
	for _, ref := range links {
		if ref.State == ast.RefStateZettelSelf {
			continue
		}
		if ref.IsZettel() {
			zid, err := id.Parse(ref.URL.Path)
			if err != nil {
				panic(err)
			}
			title, found := getTitle(zid, "html")
			if found >= 0 {
				if len(title) == 0 {
					title = ref.Value
				}
				var u string
				if found == 1 {
					ub := adapter.NewURLBuilder('h').SetZid(zid)
					if fragment := ref.URL.EscapedFragment(); len(fragment) > 0 {
						ub.SetFragment(fragment)
					}
					u = ub.String()
				}
				zetLinks = append(zetLinks, zettelReference{zid, title, len(u) > 0, u})
			}
		} else if ref.IsExternal() {
			extLinks = append(extLinks, ref.String())
		} else {
			locLinks = append(locLinks, ref.String())
		}
	}
	return zetLinks, locLinks, extLinks
}
