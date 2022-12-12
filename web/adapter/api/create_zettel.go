//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package api

import (
	"bytes"
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/content"
)

// MakePostCreatePlainZettelHandler creates a new HTTP handler to store content of
// an existing zettel.
func (a *API) MakePostCreatePlainZettelHandler(createZettel *usecase.CreateZettel) http.HandlerFunc {
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
		h := adapter.PrepareHeader(w, content.PlainText)
		h.Set(api.HeaderLocation, u)
		w.WriteHeader(http.StatusCreated)
		_, err = w.Write(newZid.Bytes())
		a.log.IfErr(err).Zid(newZid).Msg("Create Plain Zettel")
	}
}

// MakePostCreateZettelHandler creates a new HTTP handler to store content of
// an existing zettel.
func (a *API) MakePostCreateZettelHandler(createZettel *usecase.CreateZettel) http.HandlerFunc {
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
		var buf bytes.Buffer
		err = encodeJSONData(&buf, api.ZidJSON{ID: api.ZettelID(newZid.String())})
		if err != nil {
			a.log.Fatal().Err(err).Zid(newZid).Msg("Unable to store new Zid in buffer")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		h := adapter.PrepareHeader(w, content.JSON)
		h.Set(api.HeaderLocation, a.NewURLBuilder('j').SetZid(api.ZettelID(newZid.String())).String())
		w.WriteHeader(http.StatusCreated)
		_, err = w.Write(buf.Bytes())
		a.log.IfErr(err).Zid(newZid).Msg("Create JSON Zettel")
	}
}
