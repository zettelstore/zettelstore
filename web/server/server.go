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

// Server is a HTTP server.
type Server struct {
	*http.Server
	waitStop chan struct{}
}

// New creates a new HTTP server object.
func New(addr string, handler http.Handler) *Server {
	if addr == "" {
		addr = ":http"
	}
	srv := &Server{
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
func (srv *Server) SetDebug() {
	srv.ReadTimeout = 0
	srv.WriteTimeout = 0
	srv.IdleTimeout = 0
}

// Run starts the web server, but does not wait for its completion.
func (srv *Server) Run() error {
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}

	go func() { srv.Serve(ln) }()
	return nil
}

// Stop the web server.
func (srv *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	return srv.Shutdown(ctx)
}
