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
	"fmt"
	"log"
	"net/http"
	"strings"

	"zettelstore.de/z/encoder"
)

// MakePostLoginHandler creates a new HTTP handler to authenticate the given user.
func MakePostLoginHandler(apiHandler, htmlHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch format := GetFormat(r, r.URL.Query(), encoder.GetDefaultFormat()); format {
		case "json":
			apiHandler(w, r)
		case "html":
			htmlHandler(w, r)
		default:
			BadRequest(w, fmt.Sprintf("Authentication not available in format %q", format))
		}
	}
}

// GetCredentialsViaForm retrieves the authentication credentions from a form.
func GetCredentialsViaForm(r *http.Request) (ident, cred string, ok bool) {
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		return "", "", false
	}

	ident = strings.TrimSpace(r.PostFormValue("username"))
	cred = r.PostFormValue("password")
	if ident == "" {
		return "", "", false
	}
	return ident, cred, true
}
