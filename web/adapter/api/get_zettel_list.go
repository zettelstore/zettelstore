//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
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
	"fmt"
	"net/http"

	zsapi "zettelstore.de/c/api"
	"zettelstore.de/z/config"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeListMetaHandler creates a new HTTP handler for the use case "list some zettel".
func MakeListMetaHandler(listMeta usecase.ListMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		q := r.URL.Query()
		s := adapter.GetSearch(q)
		metaList, err := listMeta.Run(ctx, s)
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}

		result := make([]zsapi.ZidMetaJSON, 0, len(metaList))
		for _, m := range metaList {
			result = append(result, zsapi.ZidMetaJSON{
				ID:   zsapi.ZettelID(m.Zid.String()),
				Meta: m.Map(),
			})
		}

		w.Header().Set(zsapi.HeaderContentType, ctJSON)
		err = encodeJSONData(w, zsapi.ZettelListJSON{
			List: result,
		})
		if err != nil {
			adapter.InternalServerError(w, "Write Zettel list JSON", err)
		}
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
			adapter.ReportUsecaseError(w, err)
			return
		}

		w.Header().Set(zsapi.HeaderContentType, ctPlainText)
		for _, m := range metaList {
			_, err = fmt.Fprintln(w, m.Zid.String(), config.GetTitle(m, a.rtConfig))
			if err != nil {
				break
			}
		}
		if err != nil {
			adapter.InternalServerError(w, "Write Zettel list plain", err)
		}
	}
}
