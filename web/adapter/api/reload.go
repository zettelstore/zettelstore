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

	"zettelstore.de/z/web/adapter"
)

// ReloadHandlerAPI creates a new HTTP handler for the use case "reload".
func ReloadHandlerAPI(w http.ResponseWriter, r *http.Request, format string) {
	w.Header().Set(adapter.ContentType, format2ContentType(format))
	w.WriteHeader(http.StatusNoContent)
}
