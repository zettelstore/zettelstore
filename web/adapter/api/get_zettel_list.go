//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package api provides api handlers for web requests.
package api

import (
	"fmt"
	"net/http"

	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeListMetaHandler creates a new HTTP handler for the use case "list some zettel".
func MakeListMetaHandler(
	listMeta usecase.ListMeta,
	getMeta usecase.GetMeta,
	parseZettel usecase.ParseZettel,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filter, sorter := adapter.GetFilterSorter(q, false)
		metaList, err := listMeta.Run(r.Context(), filter, sorter)
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}

		format := adapter.GetFormat(r, q, encoder.GetDefaultFormat())
		part := getPart(q, "meta")
		w.Header().Set("Content-Type", format2ContentType(format))
		switch format {
		case "html":
			renderListMetaHTML(w, metaList)
		case "json", "djson":
			renderListMetaXJSON(r.Context(), w, metaList, format, part, getMeta, parseZettel)
		case "native", "raw", "text", "zmk":
			adapter.NotImplemented(w, fmt.Sprintf("Zettel list in format %q not yet implemented", format))
		default:
			adapter.BadRequest(w, fmt.Sprintf("Zettel list not available in format %q", format))
		}
	}
}

func renderListMetaHTML(w http.ResponseWriter, metaList []*meta.Meta) {
	buf := encoder.NewBufWriter(w)

	buf.WriteStrings("<html lang=\"", runtime.GetDefaultLang(), "\">\n<body>\n<ul>\n")
	for _, m := range metaList {
		title := m.GetDefault(meta.KeyTitle, "")
		htmlTitle, err := adapter.FormatInlines(parser.ParseTitle(title), "html")
		if err != nil {
			adapter.InternalServerError(w, "Format HTML inlines", err)
			return
		}
		buf.WriteStrings(
			"<li><a href=\"",
			adapter.NewURLBuilder('z').SetZid(m.Zid).AppendQuery("format", "html").String(),
			"\">",
			htmlTitle,
			"</a></li>\n")
	}
	buf.WriteString("</ul>\n</body>\n</html>")
	buf.Flush()
}
