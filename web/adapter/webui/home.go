//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package webui provides web-UI handlers for web requests.
package webui

import (
	"context"
	"net/http"

	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
)

type getRootStore interface {
	// GetMeta retrieves just the meta data of a specific zettel.
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
}

// MakeGetRootHandler creates a new HTTP handler to show the root URL.
func (wui *WebUI) MakeGetRootHandler(s getRootStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if r.URL.Path != "/" {
			wui.reportError(ctx, w, place.ErrNotFound)
			return
		}
		homeZid := runtime.GetHomeZettel()
		if homeZid != id.DefaultHomeZid {
			if _, err := s.GetMeta(ctx, homeZid); err == nil {
				redirectFound(w, r, wui.newURLBuilder('h').SetZid(homeZid))
				return
			}
			homeZid = id.DefaultHomeZid
		}
		_, err := s.GetMeta(ctx, homeZid)
		if err == nil {
			redirectFound(w, r, wui.newURLBuilder('h').SetZid(homeZid))
			return
		}
		if place.IsErrNotAllowed(err) && wui.authz.WithAuth() && wui.getUser(ctx) == nil {
			redirectFound(w, r, wui.newURLBuilder('a'))
			return
		}
		redirectFound(w, r, wui.newURLBuilder('h'))
	}
}
