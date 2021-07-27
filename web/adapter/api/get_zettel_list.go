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
	"net/http"

	zsapi "zettelstore.de/z/api"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeListMetaHandler creates a new HTTP handler for the use case "list some zettel".
func (api *API) MakeListMetaHandler(
	listMeta usecase.ListMeta,
	getMeta usecase.GetMeta,
	parseZettel usecase.ParseZettel,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		q := r.URL.Query()
		s := adapter.GetSearch(q, false)
		metaList, err := listMeta.Run(ctx, s)
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}

		result := make([]zsapi.ZidMetaJSON, 0, len(metaList))
		for _, m := range metaList {
			result = append(result, zsapi.ZidMetaJSON{
				ID:   m.Zid.String(),
				URL:  api.NewURLBuilder('z').SetZid(m.Zid).String(),
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
