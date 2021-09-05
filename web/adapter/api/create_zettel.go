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
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakePostCreatePlainZettelHandler creates a new HTTP handler to store content of
// an existing zettel.
func (a *API) MakePostCreatePlainZettelHandler(createZettel usecase.CreateZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zettel, err := buildZettelFromPlainData(r, id.Invalid)
		if err != nil {
			adapter.ReportUsecaseError(w, adapter.NewErrBadRequest(err.Error()))
			return
		}

		newZid, err := createZettel.Run(ctx, zettel)
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}
		u := a.NewURLBuilder('p').SetZid(newZid).String()
		h := w.Header()
		h.Set(zsapi.HeaderContentType, ctPlainText)
		h.Set(zsapi.HeaderLocation, u)
		w.WriteHeader(http.StatusCreated)
		if _, err = w.Write(newZid.Bytes()); err != nil {
			adapter.InternalServerError(w, "Write Plain", err)
		}
	}
}

// MakePostCreateZettelHandler creates a new HTTP handler to store content of
// an existing zettel.
func (a *API) MakePostCreateZettelHandler(createZettel usecase.CreateZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zettel, err := buildZettelFromJSONData(r, id.Invalid)
		if err != nil {
			adapter.ReportUsecaseError(w, adapter.NewErrBadRequest(err.Error()))
			return
		}

		newZid, err := createZettel.Run(ctx, zettel)
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}
		u := a.NewURLBuilder('j').SetZid(newZid).String()
		h := w.Header()
		h.Set(zsapi.HeaderContentType, ctJSON)
		h.Set(zsapi.HeaderLocation, u)
		w.WriteHeader(http.StatusCreated)
		if err = encodeJSONData(w, zsapi.ZidJSON{ID: newZid.String()}); err != nil {
			adapter.InternalServerError(w, "Write JSON", err)
		}
	}
}
