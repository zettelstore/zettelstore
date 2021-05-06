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

	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/router"
	"zettelstore.de/z/web/session"
)

// MakeGetDeleteZettelHandler creates a new HTTP handler to display the
// HTML delete view of a zettel.
func MakeGetDeleteZettelHandler(
	te *TemplateEngine,
	getZettel usecase.GetZettel,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if format := adapter.GetFormat(r, r.URL.Query(), "html"); format != "html" {
			te.reportError(ctx, w, adapter.NewErrBadRequest(
				fmt.Sprintf("Delete zettel not possible in format %q", format)))
			return
		}

		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			te.reportError(ctx, w, place.ErrNotFound)
			return
		}

		zettel, err := getZettel.Run(ctx, zid)
		if err != nil {
			te.reportError(ctx, w, err)
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
func MakePostDeleteZettelHandler(te *TemplateEngine, deleteZettel usecase.DeleteZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			te.reportError(ctx, w, place.ErrNotFound)
			return
		}

		if err := deleteZettel.Run(r.Context(), zid); err != nil {
			te.reportError(ctx, w, err)
			return
		}
		builder := router.GetURLBuilderFunc(ctx)
		redirectFound(w, r, builder('/'))
	}
}
