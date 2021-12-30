//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package api

import (
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/z/usecase"
)

// MakePostCommandHandler creates a new HTTP handler to execute certain commands.
func (a *API) MakePostCommandHandler(ucRefresh *usecase.Refresh) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		q := r.URL.Query()
		cmd := q.Get(api.QueryKeyCommand)
		switch api.Command(cmd) {
		case api.CommandRefresh:
			err := ucRefresh.Run(ctx)
			if err != nil {
				a.reportUsecaseError(w, err)
				return
			}
		default:
			http.Error(w, "Unknown command", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
