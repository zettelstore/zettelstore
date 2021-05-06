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
	"strings"

	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/router"
	"zettelstore.de/z/web/session"
)

// MakeGetRenameZettelHandler creates a new HTTP handler to display the
// HTML rename view of a zettel.
func MakeGetRenameZettelHandler(
	te *TemplateEngine, getMeta usecase.GetMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			te.reportError(ctx, w, place.ErrNotFound)
			return
		}

		m, err := getMeta.Run(ctx, zid)
		if err != nil {
			te.reportError(ctx, w, err)
			return
		}

		if format := adapter.GetFormat(r, r.URL.Query(), "html"); format != "html" {
			te.reportError(ctx, w, adapter.NewErrBadRequest(
				fmt.Sprintf("Rename zettel %q not possible in format %q", zid.String(), format)))
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
func MakePostRenameZettelHandler(te *TemplateEngine, renameZettel usecase.RenameZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		curZid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			te.reportError(ctx, w, place.ErrNotFound)
			return
		}

		if err = r.ParseForm(); err != nil {
			te.reportError(ctx, w, adapter.NewErrBadRequest("Unable to read rename zettel form"))
			return
		}
		if formCurZid, err1 := id.Parse(
			r.PostFormValue("curzid")); err1 != nil || formCurZid != curZid {
			te.reportError(ctx, w, adapter.NewErrBadRequest("Invalid value for current zettel id in form"))
			return
		}
		newZid, err := id.Parse(strings.TrimSpace(r.PostFormValue("newzid")))
		if err != nil {
			te.reportError(ctx, w, adapter.NewErrBadRequest(fmt.Sprintf("Invalid new zettel id %q", newZid)))
			return
		}

		if err := renameZettel.Run(r.Context(), curZid, newZid); err != nil {
			te.reportError(ctx, w, err)
			return
		}
		builder := router.GetURLBuilderFunc(ctx)
		redirectFound(w, r, builder('h').SetZid(newZid))
	}
}
