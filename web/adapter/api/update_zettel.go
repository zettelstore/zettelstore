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

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeUpdatePlainZettelHandler creates a new HTTP handler to update a zettel.
func MakeUpdatePlainZettelHandler(updateZettel usecase.UpdateZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		zettel, err := buildZettelFromPlainData(r, zid)
		if err != nil {
			adapter.ReportUsecaseError(w, adapter.NewErrBadRequest(err.Error()))
			return
		}
		if err = updateZettel.Run(r.Context(), zettel, true); err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// MakeUpdateZettelHandler creates a new HTTP handler to update a zettel.
func MakeUpdateZettelHandler(updateZettel usecase.UpdateZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		zettel, err := buildZettelFromJSONData(r, zid)
		if err != nil {
			adapter.ReportUsecaseError(w, adapter.NewErrBadRequest(err.Error()))
			return
		}
		if err = updateZettel.Run(r.Context(), zettel, true); err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
