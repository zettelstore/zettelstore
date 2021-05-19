//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
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
	"context"
	"net/http"

	"zettelstore.de/z/auth"
	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/server"
)

// MakeGetLoginHandler creates a new HTTP handler to display the HTML login view.
func MakeGetLoginHandler(auth server.Auth, te *TemplateEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderLoginForm(auth.ClearToken(r.Context(), w), w, te, false)
	}
}

func renderLoginForm(ctx context.Context, w http.ResponseWriter, te *TemplateEngine, retry bool) {
	var base baseData
	te.makeBaseData(ctx, runtime.GetDefaultLang(), "Login", nil, &base)
	te.renderTemplate(ctx, w, id.LoginTemplateZid, &base, struct {
		Title string
		Retry bool
	}{
		Title: base.Title,
		Retry: retry,
	})
}

// MakePostLoginHandlerHTML creates a new HTTP handler to authenticate the given user.
func MakePostLoginHandlerHTML(ab server.AuthBuilder, authz auth.AuthzManager, te *TemplateEngine, ucAuth usecase.Authenticate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authz.WithAuth() {
			redirectFound(w, r, ab.NewURLBuilder('/'))
			return
		}
		htmlDur, _ := startup.TokenLifetime()
		ctx := r.Context()
		ident, cred, ok := adapter.GetCredentialsViaForm(r)
		if !ok {
			te.reportError(ctx, w, adapter.NewErrBadRequest("Unable to read login form"))
			return
		}
		token, err := ucAuth.Run(ctx, ident, cred, htmlDur, auth.KindHTML)
		if err != nil {
			te.reportError(ctx, w, err)
			return
		}
		if token == nil {
			renderLoginForm(ab.ClearToken(ctx, w), w, te, true)
			return
		}

		ab.SetToken(w, token, htmlDur)
		redirectFound(w, r, ab.NewURLBuilder('/'))
	}
}

// MakeGetLogoutHandler creates a new HTTP handler to log out the current user
func MakeGetLogoutHandler(ab server.AuthBuilder, te *TemplateEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ab.ClearToken(r.Context(), w)
		redirectFound(w, r, ab.NewURLBuilder('/'))
	}
}
