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
	"encoding/json"
	"net/http"
	"time"

	"zettelstore.de/c/api"
	"zettelstore.de/z/auth"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakePostLoginHandler creates a new HTTP handler to authenticate the given user via API.
func (a *API) MakePostLoginHandler(ucAuth usecase.Authenticate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.withAuth() {
			w.Header().Set(api.HeaderContentType, ctJSON)
			writeJSONToken(w, "freeaccess", 24*366*10*time.Hour)
			return
		}
		var token []byte
		if ident, cred := retrieveIdentCred(r); ident != "" {
			var err error
			token, err = ucAuth.Run(r.Context(), ident, cred, a.tokenLifetime, auth.KindJSON)
			if err != nil {
				adapter.ReportUsecaseError(w, err)
				return
			}
		}
		if len(token) == 0 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="Default"`)
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
			return
		}

		w.Header().Set(api.HeaderContentType, ctJSON)
		writeJSONToken(w, string(token), a.tokenLifetime)
	}
}

func retrieveIdentCred(r *http.Request) (string, string) {
	if ident, cred, ok := adapter.GetCredentialsViaForm(r); ok {
		return ident, cred
	}
	if ident, cred, ok := r.BasicAuth(); ok {
		return ident, cred
	}
	return "", ""
}

func writeJSONToken(w http.ResponseWriter, token string, lifetime time.Duration) {
	je := json.NewEncoder(w)
	je.Encode(api.AuthJSON{
		Token:   token,
		Type:    "Bearer",
		Expires: int(lifetime / time.Second),
	})
}

// MakeRenewAuthHandler creates a new HTTP handler to renew the authenticate of a user.
func (a *API) MakeRenewAuthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authData := a.getAuthData(ctx)
		if authData == nil || len(authData.Token) == 0 || authData.User == nil {
			adapter.BadRequest(w, "Not authenticated")
			return
		}
		totalLifetime := authData.Expires.Sub(authData.Issued)
		currentLifetime := authData.Now.Sub(authData.Issued)
		// If we are in the first quarter of the tokens lifetime, return the token
		if currentLifetime*4 < totalLifetime {
			w.Header().Set(api.HeaderContentType, ctJSON)
			writeJSONToken(w, string(authData.Token), totalLifetime-currentLifetime)
			return
		}

		// Token is a little bit aged. Create a new one
		token, err := a.getToken(authData.User)
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}
		w.Header().Set(api.HeaderContentType, ctJSON)
		writeJSONToken(w, string(token), a.tokenLifetime)
	}
}
