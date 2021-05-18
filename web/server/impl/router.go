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
	"net/http"
	"regexp"

	"zettelstore.de/z/web/server"
)

type (
	methodHandler map[string]http.Handler
	routingTable  map[byte]methodHandler
)

// Router handles all routing for zettelstore.
type Router struct {
	urlPrefix string
	minKey    byte
	maxKey    byte
	reURL     *regexp.Regexp
	tables    [2]routingTable
	mux       *http.ServeMux
}

const (
	indexList   = 0
	indexZettel = 1
)

// NewRouter creates a new, empty router with the given root handler.
func NewRouter(urlPrefix string) *Router {
	router := &Router{
		urlPrefix: urlPrefix,
		minKey:    255,
		maxKey:    0,
		reURL:     regexp.MustCompile("^$"),
		mux:       http.NewServeMux(),
	}
	router.tables[indexList] = make(routingTable)
	router.tables[indexZettel] = make(routingTable)
	return router
}

func (rt *Router) addRoute(key byte, httpMethod string, handler http.Handler, index int) {
	// Set minKey and maxKey; re-calculate regexp.
	if key < rt.minKey || rt.maxKey < key {
		if key < rt.minKey {
			rt.minKey = key
		}
		if rt.maxKey < key {
			rt.maxKey = key
		}
		rt.reURL = regexp.MustCompile(
			"^/(?:([" + string(rt.minKey) + "-" + string(rt.maxKey) + "])(?:/(?:([0-9]{14})/?)?)?)$")
	}

	mh, hasKey := rt.tables[index][key]
	if !hasKey {
		mh = make(methodHandler)
		rt.tables[index][key] = mh
	}
	mh[httpMethod] = handler
	if httpMethod == http.MethodGet {
		if _, hasHead := rt.tables[index][key][http.MethodHead]; !hasHead {
			rt.tables[index][key][http.MethodHead] = handler
		}
	}
}

// AddListRoute adds a route for the given key and HTTP method to work with a list.
func (rt *Router) AddListRoute(key byte, httpMethod string, handler http.Handler) {
	rt.addRoute(key, httpMethod, handler, indexList)
}

// AddZettelRoute adds a route for the given key and HTTP method to work with a zettel.
func (rt *Router) AddZettelRoute(key byte, httpMethod string, handler http.Handler) {
	rt.addRoute(key, httpMethod, handler, indexZettel)
}

// NewURLBuilder creates a new URL builder.
func (rt *Router) NewURLBuilder(key byte) server.URLBuilder {
	return &URLBuilder{router: rt, key: key}
}

// URLBuilderFunc creates a new URLBuilder.
type URLBuilderFunc func(key byte) server.URLBuilder

type ctxKeyTypeRouter struct{}

var ctxKeyRouter ctxKeyTypeRouter

func (rt *Router) updateRequest(r *http.Request) *http.Request {
	ctx := r.Context()
	return r.WithContext(context.WithValue(ctx, ctxKeyRouter, rt))
}

// GetURLPrefix returns the URL prefix.
func GetURLPrefix(ctx context.Context) string {
	if rt, ok := ctx.Value(ctxKeyRouter).(*Router); ok {
		return rt.urlPrefix
	}
	return "/"
}

// GetURLBuilderFunc returns a function that creates an URL builder.
func GetURLBuilderFunc(ctx context.Context) URLBuilderFunc {
	if rt, ok := ctx.Value(ctxKeyRouter).(*Router); ok {
		return rt.NewURLBuilder
	}
	return nil
}

// Handle registers the handler for the given pattern. If a handler already exists for pattern, Handle panics.
func (rt *Router) Handle(pattern string, handler http.Handler) {
	rt.mux.Handle(pattern, handler)
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if prefixLen := len(rt.urlPrefix); prefixLen > 1 {
		if len(r.URL.Path) < prefixLen || r.URL.Path[:prefixLen] != rt.urlPrefix {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		r.URL.Path = r.URL.Path[prefixLen-1:]
	}
	match := rt.reURL.FindStringSubmatch(r.URL.Path)
	if len(match) == 3 {
		key := match[1][0]
		index := indexZettel
		if match[2] == "" {
			index = indexList
		}
		if mh, ok := rt.tables[index][key]; ok {
			if handler, ok := mh[r.Method]; ok {
				r.URL.Path = "/" + match[2]
				handler.ServeHTTP(w, rt.updateRequest(r))
				return
			}
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	}
	rt.mux.ServeHTTP(w, rt.updateRequest(r))
}