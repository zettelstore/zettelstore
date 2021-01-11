//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package adapter provides handlers for web requests.
package adapter

import (
	"net/http"

	"zettelstore.de/z/encoder"
	"zettelstore.de/z/usecase"
)

// MakeReloadHandler creates a new HTTP handler for the use case "reload".
func MakeReloadHandler(
	reload usecase.Reload,
	apiHandler func(http.ResponseWriter, *http.Request, string),
	htmlHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := reload.Run(r.Context())
		if err != nil {
			ReportUsecaseError(w, err)
			return
		}

		if format := GetFormat(r, r.URL.Query(), encoder.GetDefaultFormat()); format != "html" {
			apiHandler(w, r, format)
		}
		htmlHandler(w, r)
	}
}
