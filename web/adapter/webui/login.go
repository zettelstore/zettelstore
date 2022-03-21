//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package webui

import (
	"context"
	"net/http"

	"zettelstore.de/z/auth"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetLoginOutHandler creates a new HTTP handler to display the HTML login view,
// or to execute a logout.
func (wui *WebUI) MakeGetLoginOutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Has("logout") {
			wui.clearToken(r.Context(), w)
			wui.redirectFound(w, r, wui.NewURLBuilder('/'))
			return
		}
		wui.renderLoginForm(wui.clearToken(r.Context(), w), w, false)
	}
}

func (wui *WebUI) renderLoginForm(ctx context.Context, w http.ResponseWriter, retry bool) {
	var base baseData
	wui.makeBaseData(ctx, wui.rtConfig.GetDefaultLang(), "Login", "", nil, &base)
	wui.renderTemplate(ctx, w, id.LoginTemplateZid, &base, struct {
		Title string
		Retry bool
	}{
		Title: base.Title,
		Retry: retry,
	})
}

// MakePostLoginHandler creates a new HTTP handler to authenticate the given user.
func (wui *WebUI) MakePostLoginHandler(ucAuth *usecase.Authenticate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !wui.authz.WithAuth() {
			wui.redirectFound(w, r, wui.NewURLBuilder('/'))
			return
		}
		ctx := r.Context()
		ident, cred, ok := adapter.GetCredentialsViaForm(r)
		if !ok {
			wui.reportError(ctx, w, adapter.NewErrBadRequest("Unable to read login form"))
			return
		}
		token, err := ucAuth.Run(ctx, ident, cred, wui.tokenLifetime, auth.KindHTML)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		if token == nil {
			wui.renderLoginForm(wui.clearToken(ctx, w), w, true)
			return
		}

		wui.setToken(w, token)
		wui.redirectFound(w, r, wui.NewURLBuilder('/'))
	}
}
