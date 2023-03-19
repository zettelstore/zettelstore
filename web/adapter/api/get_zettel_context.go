//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package api

import (
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeZettelContextHandler creates a new HTTP handler for the use case "zettel context".
func (a *API) MakeZettelContextHandler(getContext usecase.ZettelContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		dir := adapter.GetZCDirection(q.Get(api.QueryKeyDir))
		cost, ok := adapter.GetInteger(q, api.QueryKeyCost)
		if !ok || cost < 0 {
			cost = 17
		}
		limit, ok := adapter.GetInteger(q, api.QueryKeyLimit)
		if !ok || limit < 0 {
			limit = 200
		}
		ctx := r.Context()
		metaList, err := getContext.Run(ctx, zid, dir, cost, limit)
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}

		err = a.writeMetaList(ctx, w, metaList[0], metaList[1:])
		a.log.IfErr(err).Zid(zid).Msg("Write Context")
	}
}
