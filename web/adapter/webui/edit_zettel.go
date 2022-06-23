//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package webui

import (
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeEditGetZettelHandler creates a new HTTP handler to display the
// HTML edit view of a zettel.
func (wui *WebUI) MakeEditGetZettelHandler(getZettel usecase.GetZettel, ucListRole usecase.ListRole) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			wui.reportError(ctx, w, box.ErrNotFound)
			return
		}

		zettel, err := getZettel.Run(box.NoEnrichContext(ctx), zid)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}

		roleData := retrieveDataLists(ctx, ucListRole)
		wui.renderZettelForm(ctx, w, zettel, "Edit Zettel", "Edit Zettel", roleData)
	}
}

// MakeEditSetZettelHandler creates a new HTTP handler to store content of
// an existing zettel.
func (wui *WebUI) MakeEditSetZettelHandler(updateZettel *usecase.UpdateZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			wui.reportError(ctx, w, box.ErrNotFound)
			return
		}

		reEdit, zettel, hasContent, err := parseZettelForm(r, zid)
		if err != nil {
			wui.reportError(ctx, w, adapter.NewErrBadRequest("Unable to read zettel form"))
			return
		}

		if err = updateZettel.Run(r.Context(), zettel, hasContent); err != nil {
			wui.reportError(ctx, w, err)
			return
		}

		if reEdit {
			wui.redirectFound(w, r, wui.NewURLBuilder('e').SetZid(api.ZettelID(zid.String())))
		} else {
			wui.redirectFound(w, r, wui.NewURLBuilder('h').SetZid(api.ZettelID(zid.String())))
		}
	}
}
