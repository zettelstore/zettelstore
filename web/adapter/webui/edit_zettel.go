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
)

// MakeEditGetZettelHandler creates a new HTTP handler to display the
// HTML edit view of a zettel.
func (wui *WebUI) MakeEditGetZettelHandler(getZettel usecase.GetZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			wui.te.reportError(ctx, w, place.ErrNotFound)
			return
		}

		zettel, err := getZettel.Run(place.NoEnrichContext(ctx), zid)
		if err != nil {
			wui.te.reportError(ctx, w, err)
			return
		}

		if format := adapter.GetFormat(r, r.URL.Query(), "html"); format != "html" {
			wui.te.reportError(ctx, w, adapter.NewErrBadRequest(
				fmt.Sprintf("Edit zettel %q not possible in format %q", zid, format)))
			return
		}

		user := wui.ab.GetUser(ctx)
		m := zettel.Meta
		var base baseData
		wui.te.makeBaseData(ctx, runtime.GetLang(m), "Edit Zettel", user, &base)
		wui.te.renderTemplate(ctx, w, id.FormTemplateZid, &base, formZettelData{
			Heading:       base.Title,
			MetaTitle:     m.GetDefault(meta.KeyTitle, ""),
			MetaRole:      m.GetDefault(meta.KeyRole, ""),
			MetaTags:      m.GetDefault(meta.KeyTags, ""),
			MetaSyntax:    m.GetDefault(meta.KeySyntax, ""),
			MetaPairsRest: m.PairsRest(false),
			IsTextContent: !zettel.Content.IsBinary(),
			Content:       zettel.Content.AsString(),
		})
	}
}

// MakeEditSetZettelHandler creates a new HTTP handler to store content of
// an existing zettel.
func (wui *WebUI) MakeEditSetZettelHandler(updateZettel usecase.UpdateZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			wui.te.reportError(ctx, w, place.ErrNotFound)
			return
		}

		zettel, hasContent, err := parseZettelForm(r, zid)
		if err != nil {
			wui.te.reportError(ctx, w, adapter.NewErrBadRequest("Unable to read zettel form"))
			return
		}

		if err := updateZettel.Run(r.Context(), zettel, hasContent); err != nil {
			wui.te.reportError(ctx, w, err)
			return
		}
		redirectFound(w, r, wui.ab.NewURLBuilder('h').SetZid(zid))
	}
}
