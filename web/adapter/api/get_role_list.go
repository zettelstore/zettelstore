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

	"zettelstore.de/c/api"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeListRoleHandler creates a new HTTP handler for the use case "list roles".
func (a *API) MakeListRoleHandler(listRole usecase.ListRole) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleList, err := listRole.Run(r.Context())
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}

		adapter.PrepareHeader(w, ctJSON)
		w.WriteHeader(http.StatusOK)
		err = encodeJSONData(w, api.RoleListJSON{Roles: roleList})
		a.log.IfErr(err).Msg("Write Roles")
	}
}
