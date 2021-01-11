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
	"context"
	"fmt"
	"net/http"
	"time"

	"zettelstore.de/z/auth/token"
	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/session"
)

// MakeGetLoginHandler creates a new HTTP handler to display the HTML login view.
func MakeGetLoginHandler(te *TemplateEngine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		renderLoginForm(session.ClearToken(r.Context(), w), w, te, false)
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
func MakePostLoginHandlerHTML(te *TemplateEngine, auth usecase.Authenticate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !startup.WithAuth() {
			http.Redirect(w, r, adapter.NewURLBuilder('/').String(), http.StatusFound)
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
	ident, cred, ok := adapter.GetCredentialsViaForm(r)
	if !ok {
		adapter.BadRequest(w, "Unable to read login form")
		return
	}
	ctx := r.Context()
	token, err := auth.Run(ctx, ident, cred, authDuration, token.KindHTML)
	if err != nil {
		adapter.ReportUsecaseError(w, err)
		return
	}
	if token == nil {
		renderLoginForm(session.ClearToken(ctx, w), w, te, true)
		return
	}

	session.SetToken(w, token, authDuration)
	http.Redirect(w, r, adapter.NewURLBuilder('/').String(), http.StatusFound)
}

// MakeGetLogoutHandler creates a new HTTP handler to log out the current user
func MakeGetLogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if format := adapter.GetFormat(r, r.URL.Query(), "html"); format != "html" {
			adapter.BadRequest(w, fmt.Sprintf("Logout not possible in format %q", format))
			return
		}

		session.ClearToken(r.Context(), w)
		http.Redirect(w, r, adapter.NewURLBuilder('/').String(), http.StatusFound)
	}
}
