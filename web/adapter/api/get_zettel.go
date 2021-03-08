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
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/index"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetZettelHandler creates a new HTTP handler to return a rendered zettel.
func MakeGetZettelHandler(
	parseZettel usecase.ParseZettel, getMeta usecase.GetMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		q := r.URL.Query()
		format := adapter.GetFormat(r, q, encoder.GetDefaultFormat())
		if format == "raw" {
			ctx = index.NoEnrichContext(ctx)
		}
		zn, err := parseZettel.Run(ctx, zid, q.Get("syntax"))
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}

		part := getPart(q, partZettel)
		switch format {
		case "json", "djson":
			if part == partUnknown {
				adapter.BadRequest(w, "Unknown _part parameter")
				return
			}
			w.Header().Set(adapter.ContentType, format2ContentType(format))
			if format != "djson" {
				err = writeJSONZettel(w, zn, part)
			} else {
				err = writeDJSONZettel(ctx, w, zn, part, partZettel, getMeta)
			}
			if err != nil {
				adapter.InternalServerError(w, "Write D/JSON", err)
			}
			return
		}

		env := encoder.Environment{
			LinkAdapter:    adapter.MakeLinkAdapter(ctx, 'z', getMeta, part.DefString(partZettel), format),
			ImageAdapter:   adapter.MakeImageAdapter(),
			CiteAdapter:    nil,
			Lang:           runtime.GetLang(zn.InhMeta),
			Xhtml:          false,
			MarkerExternal: "",
			NewWindow:      false,
			IgnoreMeta:     map[string]bool{meta.KeyLang: true},
			Title:          nil,
		}
		switch part {
		case partZettel:
			inhMeta := false
			if format != "raw" {
				w.Header().Set(adapter.ContentType, format2ContentType(format))
				inhMeta = true
			}
			enc := encoder.Create(format, &env)
			if enc == nil {
				err = adapter.ErrNoSuchFormat
			} else {
				_, err = enc.WriteZettel(w, zn, inhMeta)
			}
		case partMeta:
			w.Header().Set(adapter.ContentType, format2ContentType(format))
			if format == "raw" {
				// Don't write inherited meta data, just the raw
				err = writeMeta(w, zn.Zettel.Meta, format, nil)
			} else {
				err = writeMeta(w, zn.InhMeta, format, nil)
			}
		case partContent:
			if format == "raw" {
				if ct, ok := syntax2contentType(runtime.GetSyntax(zn.Zettel.Meta)); ok {
					w.Header().Add(adapter.ContentType, ct)
				}
			} else {
				w.Header().Set(adapter.ContentType, format2ContentType(format))
			}
			err = writeContent(w, zn, format, &env)
		default:
			adapter.BadRequest(w, "Unknown _part parameter")
			return
		}
		if err != nil {
			if err == adapter.ErrNoSuchFormat {
				adapter.BadRequest(w, fmt.Sprintf("Zettel %q not available in format %q", zid.String(), format))
				return
			}
			adapter.InternalServerError(w, "Get zettel", err)
		}
	}
}
