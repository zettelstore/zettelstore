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

	"zettelstore.de/z/auth"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/service"
	"zettelstore.de/z/web/server"
)

// API holds all data and methods for delivering API call results.
type API struct {
	b     server.Builder
	authz auth.AuthzManager
	token auth.TokenManager
	auth  server.Auth

	tokenLifetime time.Duration
}

// New creates a new API object.
func New(b server.Builder, authz auth.AuthzManager, token auth.TokenManager, auth server.Auth) *API {
	api := &API{
		b:     b,
		authz: authz,
		token: token,
		auth:  auth,

		tokenLifetime: service.Main.GetConfig(service.SubWeb, service.WebTokenLifetimeAPI).(time.Duration),
	}
	return api
}

// GetURLPrefix returns the configured URL prefix of the web server.
func (api *API) GetURLPrefix() string { return api.b.GetURLPrefix() }

// NewURLBuilder creates a new URL builder object with the given key.
func (api *API) NewURLBuilder(key byte) server.URLBuilder { return api.b.NewURLBuilder(key) }

func (api *API) getAuthData(ctx context.Context) *server.AuthData {
	return api.auth.GetAuthData(ctx)
}
func (api *API) withAuth() bool { return api.authz.WithAuth() }
func (api *API) getToken(ident *meta.Meta) ([]byte, error) {
	return api.token.GetToken(ident, api.tokenLifetime, auth.KindJSON)
}