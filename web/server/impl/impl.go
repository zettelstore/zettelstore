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
	"context"
	"net/http"
	"time"

	"zettelstore.de/z/auth/token"
	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain/meta"
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
	srv.server.Handler = newSessionHandler(srv.router, ur)
}
func (srv *myServer) GetUser(ctx context.Context) *meta.Meta {
	if data := srv.GetAuthData(ctx); data != nil {
		return data.User
	}
	return nil
}
func (srv *myServer) NewURLBuilder(key byte) server.URLBuilder {
	return srv.router.NewURLBuilder(key)
}
func (srv *myServer) GetURLPrefix() string {
	return srv.router.urlPrefix
}

const sessionName = "zsession"

func (srv *myServer) SetToken(w http.ResponseWriter, token []byte, d time.Duration) {
	cookie := http.Cookie{
		Name:     sessionName,
		Value:    string(token),
		Path:     srv.GetURLPrefix(),
		Secure:   startup.SecureCookie(),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	if startup.PersistentCookie() && d > 0 {
		cookie.Expires = time.Now().Add(d).Add(30 * time.Second).UTC()
	}
	http.SetCookie(w, &cookie)
}

// ClearToken invalidates the session cookie by sending an empty one.
func (srv *myServer) ClearToken(ctx context.Context, w http.ResponseWriter) context.Context {
	if w != nil {
		srv.SetToken(w, nil, 0)
	}
	return updateContext(ctx, nil, nil)
}

// GetAuthData returns the full authentication data from the context.
func (srv *myServer) GetAuthData(ctx context.Context) *server.AuthData {
	data, ok := ctx.Value(ctxKeySession).(*server.AuthData)
	if ok {
		return data
	}
	return nil
}

type ctxKeyTypeSession struct{}

var ctxKeySession ctxKeyTypeSession

func updateContext(ctx context.Context, user *meta.Meta, data *token.Data) context.Context {
	if data == nil {
		return context.WithValue(ctx, ctxKeySession, &server.AuthData{User: user})
	}
	return context.WithValue(
		ctx,
		ctxKeySession,
		&server.AuthData{
			User:    user,
			Token:   data.Token,
			Now:     data.Now,
			Issued:  data.Issued,
			Expires: data.Expires,
		})
}

func (srv *myServer) SetDebug()   { srv.server.SetDebug() }
func (srv *myServer) Run() error  { return srv.server.Run() }
func (srv *myServer) Stop() error { return srv.server.Stop() }
