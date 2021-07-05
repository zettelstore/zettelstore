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
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetOrderHandler creates a new API handler to return zettel references
// of a given zettel.
func (api *API) MakeGetOrderHandler(zettelOrder usecase.ZettelOrder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		ctx := r.Context()
		q := r.URL.Query()
		start, metas, err := zettelOrder.Run(ctx, zid, q.Get(meta.KeySyntax))
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}
		api.writeMetaList(w, start, metas)
	}
}
