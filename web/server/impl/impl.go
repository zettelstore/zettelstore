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

	"zettelstore.de/z/api"
	"zettelstore.de/z/auth"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/web/server"
)

type myServer struct {
	server           httpServer
	router           httpRouter
	persistentCookie bool
	secureCookie     bool
}

// New creates a new web server.
func New(listenAddr, urlPrefix string, persistentCookie, secureCookie bool, auth auth.TokenManager) server.Server {
	srv := myServer{
		persistentCookie: persistentCookie,
		secureCookie:     secureCookie,
	}
	srv.router.initializeRouter(urlPrefix, auth)
	srv.server.initializeHTTPServer(listenAddr, &srv.router)
	return &srv
}

func (srv *myServer) Handle(pattern string, handler http.Handler) {
	srv.router.Handle(pattern, handler)
}
func (srv *myServer) AddListRoute(key byte, httpMethod string, handler http.Handler) {
	srv.router.addListRoute(key, httpMethod, handler)
}
func (srv *myServer) AddZettelRoute(key byte, httpMethod string, handler http.Handler) {
	srv.router.addZettelRoute(key, httpMethod, handler)
}
func (srv *myServer) SetUserRetriever(ur server.UserRetriever) {
	srv.router.ur = ur
}
func (srv *myServer) GetUser(ctx context.Context) *meta.Meta {
	if data := srv.GetAuthData(ctx); data != nil {
		return data.User
	}
	return nil
}
func (srv *myServer) NewURLBuilder(key byte) *api.URLBuilder {
	return api.NewURLBuilder(srv.GetURLPrefix(), key)
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
		Secure:   srv.secureCookie,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	if srv.persistentCookie && d > 0 {
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

func updateContext(ctx context.Context, user *meta.Meta, data *auth.TokenData) context.Context {
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
