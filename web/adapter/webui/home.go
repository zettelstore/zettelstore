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
	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/web/router"
	"zettelstore.de/z/web/session"
)

type getRootStore interface {
	// GetMeta retrieves just the meta data of a specific zettel.
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
}

// MakeGetRootHandler creates a new HTTP handler to show the root URL.
func MakeGetRootHandler(te *TemplateEngine, s getRootStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if r.URL.Path != "/" {
			te.reportError(ctx, w, place.ErrNotFound)
			return
		}
		builder := router.GetURLBuilderFunc(ctx)
		homeZid := runtime.GetHomeZettel()
		if homeZid != id.DefaultHomeZid {
			if _, err := s.GetMeta(ctx, homeZid); err == nil {
				redirectFound(w, r, builder('h').SetZid(homeZid))
				return
			}
			homeZid = id.DefaultHomeZid
		}
		_, err := s.GetMeta(ctx, homeZid)
		if err == nil {
			redirectFound(w, r, builder('h').SetZid(homeZid))
			return
		}
		if place.IsErrNotAllowed(err) && startup.WithAuth() && session.GetUser(ctx) == nil {
			redirectFound(w, r, builder('a'))
			return
		}
		redirectFound(w, r, builder('h'))
	}
}
