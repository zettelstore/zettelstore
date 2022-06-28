//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package api provides api handlers for web requests.
package api

import (
	"bytes"
	"fmt"
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeListMetaHandler creates a new HTTP handler for the use case "list some zettel".
func (a *API) MakeListMetaHandler(listMeta usecase.ListMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		q := r.URL.Query()
		s := adapter.GetSearch(q)
		metaList, err := listMeta.Run(ctx, s)
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}

		result := make([]api.ZidMetaJSON, 0, len(metaList))
		for _, m := range metaList {
			result = append(result, api.ZidMetaJSON{
				ID:     api.ZettelID(m.Zid.String()),
				Meta:   m.Map(),
				Rights: a.getRights(ctx, m),
			})
		}

		var buf bytes.Buffer
		err = encodeJSONData(&buf, api.ZettelListJSON{
			Query: s.String(),
			List:  result,
		})
		if err != nil {
			a.log.Fatal().Err(err).Msg("Unable to store meta list in buffer")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		err = writeBuffer(w, &buf, ctJSON)
		a.log.IfErr(err).Msg("Write JSON List")
	}
}

// MakeListPlainHandler creates a new HTTP handler for the use case "list some zettel".
func (a *API) MakeListPlainHandler(listMeta usecase.ListMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		q := r.URL.Query()
		s := adapter.GetSearch(q)
		metaList, err := listMeta.Run(ctx, s)
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}

		var buf bytes.Buffer
		for _, m := range metaList {
			_, err = fmt.Fprintln(&buf, m.Zid.String(), m.GetTitle())
			if err != nil {
				a.log.Fatal().Err(err).Msg("Unable to store plain list in buffer")
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		}

		err = writeBuffer(w, &buf, ctPlainText)
		a.log.IfErr(err).Msg("Write Plain List")
	}
}
