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

	"zettelstore.de/z/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetDeleteZettelHandler creates a new HTTP handler to display the
// HTML delete view of a zettel.
func (wui *WebUI) MakeGetDeleteZettelHandler(getZettel usecase.GetZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if format, formatText := adapter.GetFormat(r, r.URL.Query(), api.EncoderHTML); format != api.EncoderHTML {
			wui.reportError(ctx, w, adapter.NewErrBadRequest(
				fmt.Sprintf("Delete zettel not possible in format %q", formatText)))
			return
		}

		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			wui.reportError(ctx, w, box.ErrNotFound)
			return
		}

		zettel, err := getZettel.Run(ctx, zid)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}

		user := wui.getUser(ctx)
		m := zettel.Meta
		var base baseData
		wui.makeBaseData(ctx, config.GetLang(m, wui.rtConfig), "Delete Zettel "+m.Zid.String(), user, &base)
		wui.renderTemplate(ctx, w, id.DeleteTemplateZid, &base, struct {
			Zid       string
			MetaPairs []meta.Pair
		}{
			Zid:       zid.String(),
			MetaPairs: m.Pairs(true),
		})
	}
}

// MakePostDeleteZettelHandler creates a new HTTP handler to delete a zettel.
func (wui *WebUI) MakePostDeleteZettelHandler(deleteZettel usecase.DeleteZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			wui.reportError(ctx, w, box.ErrNotFound)
			return
		}

		if err := deleteZettel.Run(r.Context(), zid); err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		redirectFound(w, r, wui.NewURLBuilder('/'))
	}
}
