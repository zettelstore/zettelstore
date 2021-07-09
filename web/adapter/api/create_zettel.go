//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
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
	"net/http"

	zsapi "zettelstore.de/z/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakePostCreateZettelHandler creates a new HTTP handler to store content of
// an existing zettel.
func (api *API) MakePostCreateZettelHandler(createZettel usecase.CreateZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zettel, err := buildZettelFromData(r, id.Invalid)
		if err != nil {
			adapter.ReportUsecaseError(w, adapter.NewErrBadRequest(err.Error()))
			return
		}

		newZid, err := createZettel.Run(ctx, zettel)
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}
		u := api.NewURLBuilder('z').SetZid(newZid).String()
		h := w.Header()
		h.Set(adapter.ContentType, format2ContentType(encoder.EncoderJSON))
		h.Set(adapter.Location, u)
		w.WriteHeader(http.StatusCreated)
		if err = encodeJSONData(w, zsapi.ZidJSON{ID: newZid.String(), URL: u}); err != nil {
			adapter.InternalServerError(w, "Write JSON", err)
		}
	}
}
