//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2021-present Detlef Stern
//-----------------------------------------------------------------------------

package api

import (
	"net/http"

	"t73f.de/r/sx"
	"t73f.de/r/zsc/api"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/content"
	"zettelstore.de/z/zettel"
	"zettelstore.de/z/zettel/id"
)

// MakePostCreateZettelHandler creates a new HTTP handler to store content of
// an existing zettel.
func (a *API) MakePostCreateZettelHandler(createZettel *usecase.CreateZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		enc, encStr := getEncoding(r, q)
		var zettel zettel.Zettel
		var err error
		switch enc {
		case api.EncoderPlain:
			zettel, err = buildZettelFromPlainData(r, id.InvalidO)
		case api.EncoderData:
			zettel, err = buildZettelFromData(r, id.InvalidO)
		default:
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if err != nil {
			a.reportUsecaseError(w, adapter.NewErrBadRequest(err.Error()))
			return
		}

		ctx := r.Context()
		newZid, err := createZettel.Run(ctx, zettel)
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}

		var result []byte
		var contentType string
		location := a.NewURLBuilder('z').SetZid(newZid.ZettelID())
		switch enc {
		case api.EncoderPlain:
			result = newZid.Bytes()
			contentType = content.PlainText
		case api.EncoderData:
			result = []byte(sx.Int64(newZid).String())
			contentType = content.SXPF
		default:
			panic(encStr)
		}

		h := adapter.PrepareHeader(w, contentType)
		h.Set(api.HeaderLocation, location.String())
		w.WriteHeader(http.StatusCreated)
		if _, err = w.Write(result); err != nil {
			a.log.Error().Err(err).Zid(newZid).Msg("Create Zettel")
		}
	}
}
