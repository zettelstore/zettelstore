//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
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

	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/session"
)

// MakeGetDeleteZettelHandler creates a new HTTP handler to display the
// HTML delete view of a zettel.
func MakeGetDeleteZettelHandler(
	te *TemplateEngine,
	getZettel usecase.GetZettel,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if format := adapter.GetFormat(r, r.URL.Query(), "html"); format != "html" {
			adapter.BadRequest(w, fmt.Sprintf("Delete zettel not possible in format %q", format))
			return
		}

		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		zettel, err := getZettel.Run(ctx, zid)
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}

		user := session.GetUser(ctx)
		m := zettel.Meta
		var base baseData
		te.makeBaseData(ctx, runtime.GetLang(m), "Delete Zettel "+m.Zid.String(), user, &base)
		te.renderTemplate(ctx, w, id.DeleteTemplateZid, &base, struct {
			Zid       string
			MetaPairs []meta.Pair
		}{
			Zid:       zid.String(),
			MetaPairs: m.Pairs(true),
		})
	}
}

// MakePostDeleteZettelHandler creates a new HTTP handler to delete a zettel.
func MakePostDeleteZettelHandler(deleteZettel usecase.DeleteZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}

		if err := deleteZettel.Run(r.Context(), zid); err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}
		http.Redirect(w, r, adapter.NewURLBuilder('/').String(), http.StatusFound)
	}
}
