//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
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
	"context"
	"net"
	"net/http"
	"time"
)

// Server timeout values
const (
	shutdownTimeout = 5 * time.Second
	readTimeout     = 5 * time.Second
	writeTimeout    = 10 * time.Second
	idleTimeout     = 120 * time.Second
)

// HTTPServer is a HTTP server.
type HTTPServer struct {
	*http.Server
	waitStop chan struct{}
}

// NewHTTPServer creates a new HTTP server object.
func NewHTTPServer(addr string, handler http.Handler) *HTTPServer {
	if addr == "" {
		addr = ":http"
	}
	srv := &HTTPServer{
		Server: &http.Server{
			Addr:    addr,
			Handler: handler,

			// See: https://blog.cloudflare.com/exposing-go-on-the-internet/
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
		},
		waitStop: make(chan struct{}),
	}
	return srv
}

// SetDebug enables debugging goroutines that are started by the server.
// Basically, just the timeout values are reset. This method should be called
// before running the server.
func (srv *HTTPServer) SetDebug() {
	srv.ReadTimeout = 0
	srv.WriteTimeout = 0
	srv.IdleTimeout = 0
}

// Run starts the web server, but does not wait for its completion.
func (srv *HTTPServer) Run() error {
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}

	go func() { srv.Serve(ln) }()
	return nil
}

// Stop the web server.
func (srv *HTTPServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	return srv.Shutdown(ctx)
}
