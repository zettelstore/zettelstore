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

	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/index"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/router"
)

// MakeListMetaHandler creates a new HTTP handler for the use case "list some zettel".
func MakeListMetaHandler(
	urlPrefix string,
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
		ctx1 := ctx
		if format == "html" || (!s.HasComputedMetaKey() && (part == partID || part == partContent)) {
			ctx1 = index.NoEnrichContext(ctx1)
		}
		metaList, err := listMeta.Run(ctx1, s)
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}

		w.Header().Set(adapter.ContentType, format2ContentType(format))
		switch format {
		case "html":
			renderListMetaHTML(w, router.GetURLBuilderFunc(ctx), metaList)
		case "json", "djson":
			renderListMetaXJSON(ctx, w, metaList, urlPrefix, format, part, partMeta, getMeta, parseZettel)
		case "native", "raw", "text", "zmk":
			adapter.NotImplemented(w, fmt.Sprintf("Zettel list in format %q not yet implemented", format))
		default:
			adapter.BadRequest(w, fmt.Sprintf("Zettel list not available in format %q", format))
		}
	}
}

func renderListMetaHTML(w http.ResponseWriter, builder router.URLBuilderFunc, metaList []*meta.Meta) {
	env := encoder.Environment{Interactive: true}
	buf := encoder.NewBufWriter(w)
	buf.WriteStrings("<html lang=\"", runtime.GetDefaultLang(), "\">\n<body>\n<ul>\n")
	for _, m := range metaList {
		title := m.GetDefault(meta.KeyTitle, "")
		htmlTitle, err := adapter.FormatInlines(parser.ParseMetadata(title), "html", &env)
		if err != nil {
			adapter.InternalServerError(w, "Format HTML inlines", err)
			return
		}
		buf.WriteStrings(
			"<li><a href=\"",
			builder('z').SetZid(m.Zid).AppendQuery("_format", "html").String(),
			"\">",
			htmlTitle,
			"</a></li>\n")
	}
	buf.WriteString("</ul>\n</body>\n</html>")
	buf.Flush()
}
