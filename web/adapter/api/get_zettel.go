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
		zn, err := parseZettel.Run(ctx, zid, q.Get("syntax"))
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}

		format := adapter.GetFormat(r, q, encoder.GetDefaultFormat())
		part := getPart(q, partZettel)
		switch format {
		case "json", "djson":
			if part == partUnknown {
				adapter.BadRequest(w, "Unknown _part parameter")
				return
			}
			w.Header().Set("Content-Type", format2ContentType(format))
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

		langOption := encoder.StringOption{Key: "lang", Value: runtime.GetLang(zn.InhMeta)}
		linkAdapter := encoder.AdaptLinkOption{
			Adapter: adapter.MakeLinkAdapter(ctx, 'z', getMeta, part.DefString(partZettel), format),
		}
		imageAdapter := encoder.AdaptImageOption{Adapter: adapter.MakeImageAdapter()}

		switch part {
		case partZettel:
			inhMeta := false
			if format != "raw" {
				w.Header().Set("Content-Type", format2ContentType(format))
				inhMeta = true
			}
			enc := encoder.Create(format, &langOption,
				&linkAdapter,
				&imageAdapter,
				&encoder.StringsOption{
					Key: "no-meta",
					Value: []string{
						meta.KeyLang,
					},
				},
			)
			if enc == nil {
				err = adapter.ErrNoSuchFormat
			} else {
				_, err = enc.WriteZettel(w, zn, inhMeta)
			}
		case partMeta:
			w.Header().Set("Content-Type", format2ContentType(format))
			if format == "raw" {
				// Don't write inherited meta data, just the raw
				err = writeMeta(w, zn.Zettel.Meta, format)
			} else {
				err = writeMeta(w, zn.InhMeta, format)
			}
		case partContent:
			if format == "raw" {
				if ct, ok := syntax2contentType(runtime.GetSyntax(zn.Zettel.Meta)); ok {
					w.Header().Add("Content-Type", ct)
				}
			} else {
				w.Header().Set("Content-Type", format2ContentType(format))
			}
			err = writeContent(w, zn, format,
				&langOption,
				&encoder.StringOption{
					Key:   meta.KeyMarkerExternal,
					Value: runtime.GetMarkerExternal()},
				&linkAdapter,
				&imageAdapter,
			)
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
