//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
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

	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/place"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeListMetaHandler creates a new HTTP handler for the use case "list some zettel".
func (api *API) MakeListMetaHandler(
	listMeta usecase.ListMeta,
	getMeta usecase.GetMeta,
	parseZettel usecase.ParseZettel,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		q := r.URL.Query()
		s := adapter.GetSearch(q, false)
		format := adapter.GetFormat(r, q, encoder.GetDefaultFormat())
		part := getPart(q, partMeta)
		if part == partUnknown {
			adapter.BadRequest(w, "Unknown _part parameter")
			return
		}
		ctx1 := ctx
		if format == "html" || (!s.HasComputedMetaKey() && (part == partID || part == partContent)) {
			ctx1 = place.NoEnrichContext(ctx1)
		}
		metaList, err := listMeta.Run(ctx1, s)
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}

		w.Header().Set(adapter.ContentType, format2ContentType(format))
		switch format {
		case "html":
			api.renderListMetaHTML(w, metaList)
		case "json", "djson":
			api.renderListMetaXJSON(ctx, w, metaList, format, part, partMeta, getMeta, parseZettel)
		case "native", "raw", "text", "zmk":
			adapter.NotImplemented(w, fmt.Sprintf("Zettel list in format %q not yet implemented", format))
		default:
			adapter.BadRequest(w, fmt.Sprintf("Zettel list not available in format %q", format))
		}
	}
}

func (api *API) renderListMetaHTML(w http.ResponseWriter, metaList []*meta.Meta) {
	env := encoder.Environment{Interactive: true}
	buf := encoder.NewBufWriter(w)
	buf.WriteStrings("<html lang=\"", api.rtConfig.GetDefaultLang(), "\">\n<body>\n<ul>\n")
	for _, m := range metaList {
		title := m.GetDefault(meta.KeyTitle, "")
		htmlTitle, err := adapter.FormatInlines(parser.ParseMetadata(title), "html", &env)
		if err != nil {
			adapter.InternalServerError(w, "Format HTML inlines", err)
			return
		}
		buf.WriteStrings(
			"<li><a href=\"",
			api.NewURLBuilder('z').SetZid(m.Zid).AppendQuery("_format", "html").String(),
			"\">",
			htmlTitle,
			"</a></li>\n")
	}
	buf.WriteString("</ul>\n</body>\n</html>")
	buf.Flush()
}
