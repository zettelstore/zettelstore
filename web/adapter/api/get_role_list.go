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
	"bytes"
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/z/usecase"
)

// MakeListRoleHandler creates a new HTTP handler for the use case "list roles".
func (a *API) MakeListRoleHandler(listRole usecase.ListRoles) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleList, err := listRole.Run(r.Context())
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}
		roleList.SortByName()

		var buf bytes.Buffer
		err = encodeJSONData(&buf, api.RoleListJSON{Roles: roleList.Categories()})
		if err != nil {
			a.log.Fatal().Err(err).Msg("Unable to store role list in buffer")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		err = writeBuffer(w, &buf, ctJSON)
		a.log.IfErr(err).Msg("Write Roles")
	}
}
