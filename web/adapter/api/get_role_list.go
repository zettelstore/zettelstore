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

	zsapi "zettelstore.de/z/api"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeListRoleHandler creates a new HTTP handler for the use case "list some zettel".
func (api *API) MakeListRoleHandler(listRole usecase.ListRole) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleList, err := listRole.Run(r.Context())
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}

		format, formatText := adapter.GetFormat(r, r.URL.Query(), encoder.GetDefaultFormat())
		switch format {
		case encoder.EncoderJSON:
			w.Header().Set(adapter.ContentType, format2ContentType(format))
			encodeJSONData(w, zsapi.RoleListJSON{Roles: roleList})
		default:
			adapter.BadRequest(w, fmt.Sprintf("Role list not available in format %q", formatText))
		}

	}
}
