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
	"strings"

	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/session"
)

// MakeGetRenameZettelHandler creates a new HTTP handler to display the
// HTML rename view of a zettel.
func MakeGetRenameZettelHandler(
	te *TemplateEngine, getMeta usecase.GetMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		m, err := getMeta.Run(ctx, zid)
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}

		if format := adapter.GetFormat(r, r.URL.Query(), "html"); format != "html" {
			adapter.BadRequest(w, fmt.Sprintf("Rename zettel %q not possible in format %q", zid.String(), format))
			return
		}

		user := session.GetUser(ctx)
		var base baseData
		te.makeBaseData(ctx, runtime.GetLang(m), "Rename Zettel "+zid.String(), user, &base)
		te.renderTemplate(ctx, w, id.RenameTemplateZid, &base, struct {
			Zid       string
			MetaPairs []meta.Pair
		}{
			Zid:       zid.String(),
			MetaPairs: m.Pairs(true),
		})
	}
}

// MakePostRenameZettelHandler creates a new HTTP handler to rename an existing zettel.
func MakePostRenameZettelHandler(renameZettel usecase.RenameZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		curZid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseForm(); err != nil {
			adapter.BadRequest(w, "Unable to read rename zettel form")
			return
		}
		if formCurZid, err := id.Parse(
			r.PostFormValue("curzid")); err != nil || formCurZid != curZid {
			adapter.BadRequest(w, "Invalid value for current zettel id in form")
			return
		}
		newZid, err := id.Parse(strings.TrimSpace(r.PostFormValue("newzid")))
		if err != nil {
			adapter.BadRequest(w, fmt.Sprintf("Invalid new zettel id %q", newZid.String()))
			return
		}

		if err := renameZettel.Run(r.Context(), curZid, newZid); err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}
		http.Redirect(
			w, r, adapter.NewURLBuilder('h').SetZid(newZid).String(), http.StatusFound)
	}
}
