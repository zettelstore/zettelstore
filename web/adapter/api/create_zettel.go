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

	"zettelstore.de/c/api"
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
			a.reportUsecaseError(w, adapter.NewErrBadRequest(err.Error()))
			return
		}

		newZid, err := createZettel.Run(ctx, zettel)
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}
		u := a.NewURLBuilder('z').SetZid(api.ZettelID(newZid.String())).String()
		h := adapter.PrepareHeader(w, ctPlainText)
		h.Set(api.HeaderLocation, u)
		w.WriteHeader(http.StatusCreated)
		_, err = w.Write(newZid.Bytes())
		a.log.IfErr(err).Zid(newZid).Msg("Create Plain Zettel")
	}
}

// MakePostCreateZettelHandler creates a new HTTP handler to store content of
// an existing zettel.
func (a *API) MakePostCreateZettelHandler(createZettel usecase.CreateZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zettel, err := buildZettelFromJSONData(r, id.Invalid)
		if err != nil {
			a.reportUsecaseError(w, adapter.NewErrBadRequest(err.Error()))
			return
		}

		newZid, err := createZettel.Run(ctx, zettel)
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}
		u := a.NewURLBuilder('j').SetZid(api.ZettelID(newZid.String())).String()
		h := adapter.PrepareHeader(w, ctJSON)
		h.Set(api.HeaderLocation, u)
		w.WriteHeader(http.StatusCreated)
		err = encodeJSONData(w, api.ZidJSON{ID: api.ZettelID(newZid.String())})
		a.log.IfErr(err).Zid(newZid).Msg("Create JSON Zettel")
	}
}
