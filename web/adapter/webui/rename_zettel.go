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
	"fmt"
	"net/http"
	"strings"

	"codeberg.org/t73fde/sxpf"
	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/server"
	"zettelstore.de/z/zettel/id"
)

// MakeGetRenameZettelHandler creates a new HTTP handler to display the
// HTML rename view of a zettel.
func (wui *WebUI) MakeGetRenameZettelHandler(getMeta usecase.GetMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			wui.reportError(ctx, w, box.ErrNotFound)
			return
		}

		m, err := getMeta.Run(ctx, zid)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}

		user := server.GetUser(ctx)
		env, err := wui.createRenderEnv(
			ctx, "rename",
			wui.rtConfig.Get(ctx, nil, api.KeyLang), "Rename Zettel "+m.Zid.String(), user)
		rb := makeRenderBinder(wui.sf, env, err)
		rb.bindString("zid", sxpf.MakeString(m.Zid.String()))
		rb.bindString("incoming", wui.encodeIncoming(m, wui.makeGetTextTitle(ctx, getMeta)))
		rb.bindString("useless", retrieveUselessFiles(m))
		rb.bindString("meta-pairs", makeMetaPairs(m))
		if rb.err == nil {
			err = wui.renderSxnTemplate(ctx, w, id.RenameTemplateZid, env)
		}
		if err != nil {
			wui.reportError(ctx, w, err)
		}
	}
}

// MakePostRenameZettelHandler creates a new HTTP handler to rename an existing zettel.
func (wui *WebUI) MakePostRenameZettelHandler(renameZettel *usecase.RenameZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		curZid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			wui.reportError(ctx, w, box.ErrNotFound)
			return
		}

		if err = r.ParseForm(); err != nil {
			wui.reportError(ctx, w, adapter.NewErrBadRequest("Unable to read rename zettel form"))
			return
		}
		if formCurZid, err1 := id.Parse(
			r.PostFormValue("curzid")); err1 != nil || formCurZid != curZid {
			wui.reportError(ctx, w, adapter.NewErrBadRequest("Invalid value for current zettel id in form"))
			return
		}
		formNewZid := strings.TrimSpace(r.PostFormValue("newzid"))
		newZid, err := id.Parse(formNewZid)
		if err != nil {
			wui.reportError(
				ctx, w, adapter.NewErrBadRequest(fmt.Sprintf("Invalid new zettel id %q", formNewZid)))
			return
		}

		if err = renameZettel.Run(r.Context(), curZid, newZid); err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		wui.redirectFound(w, r, wui.NewURLBuilder('h').SetZid(api.ZettelID(newZid.String())))
	}
}
