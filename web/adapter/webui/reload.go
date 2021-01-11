//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package webui provides wet-UI handlers for web requests.
package webui

import (
	"net/http"

	"zettelstore.de/z/web/adapter"
)

// ReloadHandlerHTML creates a new HTTP handler for the use case "reload".
func ReloadHandlerHTML(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, adapter.NewURLBuilder('/').String(), http.StatusFound)
}
