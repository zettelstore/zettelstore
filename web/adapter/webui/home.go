//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package webui

import (
	"context"
	"errors"
	"net/http"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/server"
	"zettelstore.de/z/zettel"
	"zettelstore.de/z/zettel/id"
)

type getRootStore interface {
	GetZettel(ctx context.Context, zid id.Zid) (zettel.Zettel, error)
}

// MakeGetRootHandler creates a new HTTP handler to show the root URL.
func (wui *WebUI) MakeGetRootHandler(s getRootStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if p := r.URL.Path; p != "/" {
			wui.reportError(ctx, w, adapter.ErrResourceNotFound{Path: p})
			return
		}
		homeZid, _ := id.Parse(wui.rtConfig.Get(ctx, nil, config.KeyHomeZettel))
		apiHomeZid := api.ZettelID(homeZid.String())
		if homeZid != id.DefaultHomeZid {
			if _, err := s.GetZettel(ctx, homeZid); err == nil {
				wui.redirectFound(w, r, wui.NewURLBuilder('h').SetZid(apiHomeZid))
				return
			}
			homeZid = id.DefaultHomeZid
		}
		_, err := s.GetZettel(ctx, homeZid)
		if err == nil {
			wui.redirectFound(w, r, wui.NewURLBuilder('h').SetZid(apiHomeZid))
			return
		}
		if errors.Is(err, &box.ErrNotAllowed{}) && wui.authz.WithAuth() && server.GetUser(ctx) == nil {
			wui.redirectFound(w, r, wui.NewURLBuilder('i'))
			return
		}
		wui.redirectFound(w, r, wui.NewURLBuilder('h'))
	}
}
