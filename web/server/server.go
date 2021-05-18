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

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

type URLBuilder interface {
	// Clone an URLBuilder
	Clone() URLBuilder

	// SetZid sets the zettel identifier.
	SetZid(zid id.Zid) URLBuilder

	// AppendPath adds a new path element
	AppendPath(p string) URLBuilder

	// AppendQuery adds a new query parameter
	AppendQuery(key, value string) URLBuilder

	// ClearQuery removes all query parameters.
	ClearQuery() URLBuilder

	// SetFragment stores the fragment
	SetFragment(s string) URLBuilder

	// String produces a string value.
	String() string
}

// UserRetriever allows to retrieve user data based on a given zettel identifier.
type UserRetriever interface {
	GetUser(ctx context.Context, zid id.Zid, ident string) (*meta.Meta, error)
}

// Router allows to state routes for various URL paths.
type Router interface {
	Handle(pattern string, handler http.Handler)
	AddListRoute(key byte, httpMethod string, handler http.Handler)
	AddZettelRoute(key byte, httpMethod string, handler http.Handler)
	SetUserRetriever(ur UserRetriever)
}

// Auth is.
type Auth interface {
	NewURLBuilder(key byte) URLBuilder
}

// Server is the main web server for accessing Zettelstore via HTTP.
type Server interface {
	Router
	Auth

	SetDebug()
	Run() error
	Stop() error
}
