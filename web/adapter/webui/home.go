//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package webui provides wet-UI handlers for web requests.
package webui

import (
	"context"
	"net/http"

	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/session"
)

type getRootStore interface {
	// GetMeta retrieves just the meta data of a specific zettel.
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
}

// MakeGetRootHandler creates a new HTTP handler to show the root URL.
func MakeGetRootHandler(s getRootStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		ok := false
		ctx := r.Context()
		homeZid := runtime.GetHomeZettel()
		if homeZid != id.DefaultHomeZid && homeZid.IsValid() {
			if _, err := s.GetMeta(ctx, homeZid); err != nil {
				homeZid = id.DefaultHomeZid
			} else {
				ok = true
			}
		}
		if !ok {
			if _, err := s.GetMeta(ctx, homeZid); err != nil {
				if place.IsErrNotAllowed(err) && startup.WithAuth() && session.GetUser(ctx) == nil {
					http.Redirect(w, r, adapter.NewURLBuilder('a').String(), http.StatusFound)
					return
				}
				http.Redirect(w, r, adapter.NewURLBuilder('h').String(), http.StatusFound)
				return
			}
		}
		http.Redirect(w, r, adapter.NewURLBuilder('h').SetZid(homeZid).String(), http.StatusFound)
	}
}
