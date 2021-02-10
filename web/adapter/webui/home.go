//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
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
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/web/adapter"
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
		homeID := runtime.GetHomeZettel()
		if homeID != id.DefaultHomeZid && homeID.IsValid() {
			if _, err := s.GetMeta(r.Context(), homeID); err != nil {
				homeID = id.DefaultHomeZid
			}
		}
		http.Redirect(w, r, adapter.NewURLBuilder('h').SetZid(homeID).String(), http.StatusFound)
	}
}
