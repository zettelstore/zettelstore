//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package server provides the Zettelstore web service.
package server

import (
	"context"
	"net/http"
	"time"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

// UserRetriever allows to retrieve user data based on a given zettel identifier.
type UserRetriever interface {
	GetUser(ctx context.Context, zid id.Zid, ident string) (*meta.Meta, error)
}

// Method enumerates the allowed HTTP methods.
type Method uint8

// Values for method type
const (
	MethodGet Method = iota
	MethodHead
	MethodPost
	MethodPut
	MethodMove
	MethodDelete
	MethodLAST // must always be the last one
)

// Router allows to state routes for various URL paths.
type Router interface {
	Handle(pattern string, handler http.Handler)
	AddListRoute(key byte, method Method, handler http.Handler)
	AddZettelRoute(key byte, method Method, handler http.Handler)
	SetUserRetriever(ur UserRetriever)
}

// Builder allows to build new URLs for the web service.
type Builder interface {
	GetURLPrefix() string
	NewURLBuilder(key byte) *api.URLBuilder
}

// Auth is the authencation interface.
type Auth interface {
	GetUser(context.Context) *meta.Meta
	SetToken(w http.ResponseWriter, token []byte, d time.Duration)

	// ClearToken invalidates the session cookie by sending an empty one.
	ClearToken(ctx context.Context, w http.ResponseWriter) context.Context

	// GetAuthData returns the full authentication data from the context.
	GetAuthData(ctx context.Context) *AuthData
}

// AuthData stores all relevant authentication data for a context.
type AuthData struct {
	User    *meta.Meta
	Token   []byte
	Now     time.Time
	Issued  time.Time
	Expires time.Time
}

// AuthBuilder is a Builder that also allows to execute authentication functions.
type AuthBuilder interface {
	Auth
	Builder
}

// Server is the main web server for accessing Zettelstore via HTTP.
type Server interface {
	Router
	Auth
	Builder

	SetDebug()
	Run() error
	Stop() error
}
