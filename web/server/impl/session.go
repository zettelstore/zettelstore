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
	"net/http"
	"strings"

	"zettelstore.de/z/auth/token"
	"zettelstore.de/z/web/server"
)

// sessionHandler enriches the request context with optional user information.
type sessionHandler struct {
	next         http.Handler
	getUserByZid server.UserRetriever
}

// newSessionHandler creates a new handler.
func newSessionHandler(next http.Handler, getUserByZid server.UserRetriever) *sessionHandler {
	return &sessionHandler{
		next:         next,
		getUserByZid: getUserByZid,
	}
}

// ServeHTTP processes one HTTP request.
func (h *sessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	k := token.KindJSON
	t := getHeaderToken(r)
	if t == nil {
		k = token.KindHTML
		t = getSessionToken(r)
	}
	if t == nil {
		h.next.ServeHTTP(w, r)
		return
	}
	tokenData, err := token.CheckToken(t, k)
	if err != nil {
		h.next.ServeHTTP(w, r)
		return
	}
	ctx := r.Context()
	user, err := h.getUserByZid.GetUser(ctx, tokenData.Zid, tokenData.Ident)
	if err != nil {
		h.next.ServeHTTP(w, r)
		return
	}
	h.next.ServeHTTP(w, r.WithContext(updateContext(ctx, user, &tokenData)))
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
