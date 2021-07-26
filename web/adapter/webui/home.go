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
	"errors"
	"net/http"

	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
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
			wui.reportError(ctx, w, box.ErrNotFound)
			return
		}
		homeZid := wui.rtConfig.GetHomeZettel()
		if homeZid != id.DefaultHomeZid {
			if _, err := s.GetMeta(ctx, homeZid); err == nil {
				redirectFound(w, r, wui.NewURLBuilder('h').SetZid(homeZid))
				return
			}
			homeZid = id.DefaultHomeZid
		}
		_, err := s.GetMeta(ctx, homeZid)
		if err == nil {
			redirectFound(w, r, wui.NewURLBuilder('h').SetZid(homeZid))
			return
		}
		if errors.Is(err, &box.ErrNotAllowed{}) && wui.authz.WithAuth() && wui.getUser(ctx) == nil {
			redirectFound(w, r, wui.NewURLBuilder('i'))
			return
		}
		redirectFound(w, r, wui.NewURLBuilder('h'))
	}
}
