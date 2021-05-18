//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package webui provides web-UI handlers for web requests.
package webui

import (
	"net/http"

	"zettelstore.de/z/web/server"
)

func redirectFound(w http.ResponseWriter, r *http.Request, ub server.URLBuilder) {
	http.Redirect(w, r, ub.String(), http.StatusFound)
}
