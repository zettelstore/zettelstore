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
	"context"
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/z/usecase"
)

// MakePostCommandHandler creates a new HTTP handler to execute certain commands.
func (a *API) MakePostCommandHandler(
	ucIsAuth *usecase.IsAuthenticated,
	ucRefresh *usecase.Refresh,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		q := r.URL.Query()
		cmd := q.Get(api.QueryKeyCommand)
		switch api.Command(cmd) {
		case api.CommandAuthenticated:
			handleIsAuthenticated(ctx, w, ucIsAuth)
			return
		case api.CommandRefresh:
			err := ucRefresh.Run(ctx)
			if err != nil {
				a.reportUsecaseError(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "Unknown command", http.StatusBadRequest)
	}
}

func handleIsAuthenticated(ctx context.Context, w http.ResponseWriter, ucIsAuth *usecase.IsAuthenticated) {
	switch ucIsAuth.Run(ctx) {
	case usecase.IsAuthenticatedDisabled:
		w.WriteHeader(http.StatusOK)
	case usecase.IsAuthenticatedAndValid:
		w.WriteHeader(http.StatusNoContent)
	case usecase.IsAuthenticatedAndInvalid:
		w.WriteHeader(http.StatusUnauthorized)
	default:
		http.Error(w, "Unexpected result value", http.StatusInternalServerError)
	}
}
