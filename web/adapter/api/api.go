//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
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
	"context"
	"time"

	"zettelstore.de/z/api"
	"zettelstore.de/z/auth"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/web/server"
)

// API holds all data and methods for delivering API call results.
type API struct {
	b        server.Builder
	rtConfig config.Config
	authz    auth.AuthzManager
	token    auth.TokenManager
	auth     server.Auth

	tokenLifetime time.Duration
}

// New creates a new API object.
func New(b server.Builder, authz auth.AuthzManager, token auth.TokenManager, auth server.Auth, rtConfig config.Config) *API {
	a := &API{
		b:        b,
		authz:    authz,
		token:    token,
		auth:     auth,
		rtConfig: rtConfig,

		tokenLifetime: kernel.Main.GetConfig(kernel.WebService, kernel.WebTokenLifetimeAPI).(time.Duration),
	}
	return a
}

// GetURLPrefix returns the configured URL prefix of the web server.
func (a *API) GetURLPrefix() string { return a.b.GetURLPrefix() }

// NewURLBuilder creates a new URL builder object with the given key.
func (a *API) NewURLBuilder(key byte) *api.URLBuilder { return a.b.NewURLBuilder(key) }

func (a *API) getAuthData(ctx context.Context) *server.AuthData {
	return a.auth.GetAuthData(ctx)
}
func (a *API) withAuth() bool { return a.authz.WithAuth() }
func (a *API) getToken(ident *meta.Meta) ([]byte, error) {
	return a.token.GetToken(ident, a.tokenLifetime, auth.KindJSON)
}
