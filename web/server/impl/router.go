//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package impl provides the Zettelstore web service.
package impl

import (
	"io"
	"net/http"
	"regexp"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/z/auth"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/logger"
	"zettelstore.de/z/web/server"
)

type (
	methodHandler [server.MethodLAST]http.Handler
	routingTable  [256]*methodHandler
)

var mapMethod = map[string]server.Method{
	http.MethodHead:   server.MethodHead,
	http.MethodGet:    server.MethodGet,
	http.MethodPost:   server.MethodPost,
	http.MethodPut:    server.MethodPut,
	http.MethodDelete: server.MethodDelete,
	api.MethodMove:    server.MethodMove,
}

// httpRouter handles all routing for zettelstore.
type httpRouter struct {
	log         *logger.Logger
	urlPrefix   string
	auth        auth.TokenManager
	minKey      byte
	maxKey      byte
	reURL       *regexp.Regexp
	listTable   routingTable
	zettelTable routingTable
	ur          server.UserRetriever
	mux         *http.ServeMux
	maxReqSize  int64
}

// initializeRouter creates a new, empty router with the given root handler.
func (rt *httpRouter) initializeRouter(log *logger.Logger, urlPrefix string, maxRequestSize int64, auth auth.TokenManager) {
	rt.log = log
	rt.urlPrefix = urlPrefix
	rt.auth = auth
	rt.minKey = 255
	rt.maxKey = 0
	rt.reURL = regexp.MustCompile("^$")
	rt.mux = http.NewServeMux()
	rt.maxReqSize = maxRequestSize
}

func (rt *httpRouter) addRoute(key byte, method server.Method, handler http.Handler, table *routingTable) {
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

	mh := table[key]
	if mh == nil {
		mh = new(methodHandler)
		table[key] = mh
	}
	mh[method] = handler
	if method == server.MethodGet {
		if prevHandler := mh[server.MethodHead]; prevHandler == nil {
			mh[server.MethodHead] = handler
		}
	}
}

// addListRoute adds a route for the given key and HTTP method to work with a list.
func (rt *httpRouter) addListRoute(key byte, method server.Method, handler http.Handler) {
	rt.addRoute(key, method, handler, &rt.listTable)
}

// addZettelRoute adds a route for the given key and HTTP method to work with a zettel.
func (rt *httpRouter) addZettelRoute(key byte, method server.Method, handler http.Handler) {
	rt.addRoute(key, method, handler, &rt.zettelTable)
}

// Handle registers the handler for the given pattern. If a handler already exists for pattern, Handle panics.
func (rt *httpRouter) Handle(pattern string, handler http.Handler) {
	rt.mux.Handle(pattern, handler)
}

func (rt *httpRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Something may panic. Ensure a kernel log.
	defer func() {
		if reco := recover(); reco != nil {
			rt.log.Error().Str("Method", r.Method).Str("URL", r.URL.String()).Msg("Recover context")
			kernel.Main.LogRecover("Web", reco)
		}
	}()

	var withDebug bool
	if msg := rt.log.Debug(); msg.Enabled() {
		withDebug = true
		w = &traceResponseWriter{original: w}
		msg.Str("method", r.Method).Str("uri", r.RequestURI).Msg("ServeHTTP")
	}

	if prefixLen := len(rt.urlPrefix); prefixLen > 1 {
		if len(r.URL.Path) < prefixLen || r.URL.Path[:prefixLen] != rt.urlPrefix {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			if withDebug {
				rt.log.Debug().Int("sc", int64(w.(*traceResponseWriter).statusCode)).Msg("/ServeHTTP/prefix")
			}
			return
		}
		r.URL.Path = r.URL.Path[prefixLen-1:]
	}
	r.Body = http.MaxBytesReader(w, r.Body, rt.maxReqSize)
	match := rt.reURL.FindStringSubmatch(r.URL.Path)
	if len(match) != 3 {
		rt.mux.ServeHTTP(w, rt.addUserContext(r))
		if withDebug {
			rt.log.Debug().Int("sc", int64(w.(*traceResponseWriter).statusCode)).Msg("match other")
		}
		return
	}
	if withDebug {
		rt.log.Debug().Str("key", match[1]).Str("zid", match[2]).Msg("path match")
	}

	key := match[1][0]
	var mh *methodHandler
	if match[2] == "" {
		mh = rt.listTable[key]
	} else {
		mh = rt.zettelTable[key]
	}
	method, ok := mapMethod[r.Method]
	if ok && mh != nil {
		if handler := mh[method]; handler != nil {
			r.URL.Path = "/" + match[2]
			handler.ServeHTTP(w, rt.addUserContext(r))
			if withDebug {
				rt.log.Debug().Int("sc", int64(w.(*traceResponseWriter).statusCode)).Msg("/ServeHTTP")
			}
			return
		}
	}
	http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	if withDebug {
		rt.log.Debug().Int("sc", int64(w.(*traceResponseWriter).statusCode)).Msg("no match")
	}
}

func (rt *httpRouter) addUserContext(r *http.Request) *http.Request {
	if rt.ur == nil {
		return r
	}
	k := auth.KindJSON
	t := getHeaderToken(r)
	if len(t) == 0 {
		k = auth.KindHTML
		t = getSessionToken(r)
	}
	if len(t) == 0 {
		return r
	}
	tokenData, err := rt.auth.CheckToken(t, k)
	if err != nil {
		return r
	}
	ctx := r.Context()
	user, err := rt.ur.GetUser(ctx, tokenData.Zid, tokenData.Ident)
	if err != nil {
		return r
	}
	return r.WithContext(updateContext(ctx, user, &tokenData))
}

func getSessionToken(r *http.Request) []byte {
	cookie, err := r.Cookie(sessionName)
	if err != nil {
		return nil
	}
	return []byte(cookie.Value)
}

func getHeaderToken(r *http.Request) []byte {
	h := r.Header["Authorization"]
	if h == nil {
		return nil
	}

	// “Multiple message-header fields with the same field-name MAY be
	// present in a message if and only if the entire field-value for that
	// header field is defined as a comma-separated list.”
	// — “Hypertext Transfer Protocol” RFC 2616, subsection 4.2
	auth := strings.Join(h, ", ")

	const prefix = "Bearer "
	// RFC 2617, subsection 1.2 defines the scheme token as case-insensitive.
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return nil
	}
	return []byte(auth[len(prefix):])
}

type traceResponseWriter struct {
	original   http.ResponseWriter
	statusCode int
}

func (w *traceResponseWriter) Header() http.Header         { return w.original.Header() }
func (w *traceResponseWriter) Write(p []byte) (int, error) { return w.original.Write(p) }
func (w *traceResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.original.WriteHeader(statusCode)
}
func (w *traceResponseWriter) WriteString(s string) (int, error) {
	return io.WriteString(w.original, s)
}
