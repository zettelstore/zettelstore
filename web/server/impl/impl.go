//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package impl provides the Zettelstore web service.
package impl

import (
	"net/http"

	"zettelstore.de/z/web/server"
)

type myServer struct {
	server *HTTPServer
	router *Router
}

// New creates a new web server.
func New(listenAddr, urlPrefix string) server.Server {
	router := NewRouter(urlPrefix)
	srv := &myServer{
		server: NewHTTPServer(listenAddr, router),
		router: router,
	}
	return srv
}

func (srv *myServer) Handle(pattern string, handler http.Handler) {
	srv.router.Handle(pattern, handler)
}
func (srv *myServer) AddListRoute(key byte, httpMethod string, handler http.Handler) {
	srv.router.AddListRoute(key, httpMethod, handler)
}
func (srv *myServer) AddZettelRoute(key byte, httpMethod string, handler http.Handler) {
	srv.router.AddZettelRoute(key, httpMethod, handler)
}
func (srv *myServer) SetUserRetriever(ur server.UserRetriever) {
	srv.server.Handler = NewHandler(srv.router, ur)
}
func (srv *myServer) NewURLBuilder(key byte) server.URLBuilder {
	return srv.router.NewURLBuilder(key)
}
func (srv *myServer) SetDebug()   { srv.server.SetDebug() }
func (srv *myServer) Run() error  { return srv.server.Run() }
func (srv *myServer) Stop() error { return srv.server.Stop() }
