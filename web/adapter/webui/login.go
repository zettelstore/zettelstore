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
	"time"

	"zettelstore.de/z/auth"
	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/service"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetLoginHandler creates a new HTTP handler to display the HTML login view.
func (wui *WebUI) MakeGetLoginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wui.renderLoginForm(wui.ab.ClearToken(r.Context(), w), w, false)
	}
}

func (wui *WebUI) renderLoginForm(ctx context.Context, w http.ResponseWriter, retry bool) {
	var base baseData
	wui.te.makeBaseData(ctx, runtime.GetDefaultLang(), "Login", nil, &base)
	wui.te.renderTemplate(ctx, w, id.LoginTemplateZid, &base, struct {
		Title string
		Retry bool
	}{
		Title: base.Title,
		Retry: retry,
	})
}

// MakePostLoginHandlerHTML creates a new HTTP handler to authenticate the given user.
func (wui *WebUI) MakePostLoginHandlerHTML(ucAuth usecase.Authenticate) http.HandlerFunc {
	tokenLifetime := service.Main.GetConfig(service.SubWeb, service.WebTokenLifetimeHTML).(time.Duration)
	return func(w http.ResponseWriter, r *http.Request) {
		if !wui.authz.WithAuth() {
			redirectFound(w, r, wui.ab.NewURLBuilder('/'))
			return
		}
		ctx := r.Context()
		ident, cred, ok := adapter.GetCredentialsViaForm(r)
		if !ok {
			wui.te.reportError(ctx, w, adapter.NewErrBadRequest("Unable to read login form"))
			return
		}
		token, err := ucAuth.Run(ctx, ident, cred, tokenLifetime, auth.KindHTML)
		if err != nil {
			wui.te.reportError(ctx, w, err)
			return
		}
		if token == nil {
			wui.renderLoginForm(wui.ab.ClearToken(ctx, w), w, true)
			return
		}

		wui.ab.SetToken(w, token, tokenLifetime)
		redirectFound(w, r, wui.ab.NewURLBuilder('/'))
	}
}

// MakeGetLogoutHandler creates a new HTTP handler to log out the current user
func (wui *WebUI) MakeGetLogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wui.ab.ClearToken(r.Context(), w)
		redirectFound(w, r, wui.ab.NewURLBuilder('/'))
	}
}
