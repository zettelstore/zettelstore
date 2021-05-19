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

	"zettelstore.de/z/auth"
	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/server"
)

// MakePostLoginHandlerAPI creates a new HTTP handler to authenticate the given user via API.
func MakePostLoginHandlerAPI(authz auth.AuthzManager, auth usecase.Authenticate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authz.WithAuth() {
			w.Header().Set(adapter.ContentType, format2ContentType("json"))
			writeJSONToken(w, "freeaccess", 24*366*10*time.Hour)
			return
		}
		_, apiDur := startup.TokenLifetime()
		authenticateViaJSON(auth, w, r, apiDur)
	}
}

func authenticateViaJSON(
	auth usecase.Authenticate,
	w http.ResponseWriter,
	r *http.Request,
	authDuration time.Duration,
) {
	token, err := authenticateForJSON(auth, w, r, authDuration)
	if err != nil {
		adapter.ReportUsecaseError(w, err)
		return
	}
	if token == nil {
		w.Header().Set("WWW-Authenticate", `Bearer realm="Default"`)
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	w.Header().Set(adapter.ContentType, format2ContentType("json"))
	writeJSONToken(w, string(token), authDuration)
}

func authenticateForJSON(
	ucAuth usecase.Authenticate,
	w http.ResponseWriter,
	r *http.Request,
	authDuration time.Duration,
) ([]byte, error) {
	ident, cred, ok := adapter.GetCredentialsViaForm(r)
	if !ok {
		if ident, cred, ok = r.BasicAuth(); !ok {
			return nil, nil
		}
	}
	token, err := ucAuth.Run(r.Context(), ident, cred, authDuration, auth.KindJSON)
	return token, err
}

func writeJSONToken(w http.ResponseWriter, token string, lifetime time.Duration) {
	je := json.NewEncoder(w)
	je.Encode(struct {
		Token   string `json:"access_token"`
		Type    string `json:"token_type"`
		Expires int    `json:"expires_in"`
	}{
		Token:   token,
		Type:    "Bearer",
		Expires: int(lifetime / time.Second),
	})
}

// MakeRenewAuthHandler creates a new HTTP handler to renew the authenticate of a user.
func MakeRenewAuthHandler(token auth.TokenManager, authS server.Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authData := authS.GetAuthData(ctx)
		if authData == nil || authData.Token == nil || authData.User == nil {
			adapter.BadRequest(w, "Not authenticated")
			return
		}
		totalLifetime := authData.Expires.Sub(authData.Issued)
		currentLifetime := authData.Now.Sub(authData.Issued)
		// If we are in the first quarter of the tokens lifetime, return the token
		if currentLifetime*4 < totalLifetime {
			w.Header().Set(adapter.ContentType, format2ContentType("json"))
			writeJSONToken(w, string(authData.Token), totalLifetime-currentLifetime)
			return
		}

		// Toke is a little bit aged. Create a new one
		_, apiDur := startup.TokenLifetime()
		token, err := token.GetToken(authData.User, apiDur, auth.KindJSON)
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}
		w.Header().Set(adapter.ContentType, format2ContentType("json"))
		writeJSONToken(w, string(token), apiDur)
	}
}
