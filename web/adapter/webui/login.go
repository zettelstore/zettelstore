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
	"fmt"
	"net/http"
	"time"

	"zettelstore.de/z/auth"
	"zettelstore.de/z/auth/token"
	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/server"
)

// MakeGetLoginHandler creates a new HTTP handler to display the HTML login view.
func MakeGetLoginHandler(te *TemplateEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderLoginForm(server.ClearToken(r.Context(), w), w, te, false)
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
func MakePostLoginHandlerHTML(authz auth.AuthzManager, te *TemplateEngine, auth usecase.Authenticate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authz.WithAuth() {
			builder := server.GetURLBuilderFunc(r.Context())
			redirectFound(w, r, builder('/'))
			return
		}
		htmlDur, _ := startup.TokenLifetime()
		authenticateViaHTML(te, auth, w, r, htmlDur)
	}
}

func authenticateViaHTML(
	te *TemplateEngine,
	auth usecase.Authenticate,
	w http.ResponseWriter,
	r *http.Request,
	authDuration time.Duration,
) {
	ctx := r.Context()
	ident, cred, ok := adapter.GetCredentialsViaForm(r)
	if !ok {
		te.reportError(ctx, w, adapter.NewErrBadRequest("Unable to read login form"))
		return
	}
	token, err := auth.Run(ctx, ident, cred, authDuration, token.KindHTML)
	if err != nil {
		te.reportError(ctx, w, err)
		return
	}
	if token == nil {
		renderLoginForm(server.ClearToken(ctx, w), w, te, true)
		return
	}

	server.SetToken(w, token, authDuration)
	builder := server.GetURLBuilderFunc(ctx)
	redirectFound(w, r, builder('/'))
}

// MakeGetLogoutHandler creates a new HTTP handler to log out the current user
func MakeGetLogoutHandler(te *TemplateEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if format := adapter.GetFormat(r, r.URL.Query(), "html"); format != "html" {
			te.reportError(ctx, w, adapter.NewErrBadRequest(
				fmt.Sprintf("Logout not possible in format %q", format)))
			return
		}

		server.ClearToken(ctx, w)
		builder := server.GetURLBuilderFunc(ctx)
		redirectFound(w, r, builder('/'))
	}
}
