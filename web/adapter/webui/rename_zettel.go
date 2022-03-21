//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
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

	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetRenameZettelHandler creates a new HTTP handler to display the
// HTML rename view of a zettel.
func (wui *WebUI) MakeGetRenameZettelHandler(getMeta usecase.GetMeta, evaluate *usecase.Evaluate) http.HandlerFunc {
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

		if enc, encText := adapter.GetEncoding(r, r.URL.Query(), api.EncoderHTML); enc != api.EncoderHTML {
			wui.reportError(ctx, w, adapter.NewErrBadRequest(
				fmt.Sprintf("Rename zettel %q not possible in encoding %q", zid.String(), encText)))
			return
		}

		getTextTitle := wui.makeGetTextTitle(ctx, getMeta, evaluate)
		incomingLinks := wui.encodeIncoming(m, getTextTitle)
		uselessFiles := retrieveUselessFiles(m)

		user := wui.getUser(ctx)
		var base baseData
		wui.makeBaseData(ctx, config.GetLang(m, wui.rtConfig), "Rename Zettel "+zid.String(), "", user, &base)
		wui.renderTemplate(ctx, w, id.RenameTemplateZid, &base, struct {
			Zid             string
			MetaPairs       []meta.Pair
			HasIncoming     bool
			Incoming        []simpleLink
			HasUselessFiles bool
			UselessFiles    []string
		}{
			Zid:             zid.String(),
			MetaPairs:       m.ComputedPairs(),
			HasIncoming:     len(incomingLinks) > 0,
			Incoming:        incomingLinks,
			HasUselessFiles: len(uselessFiles) > 0,
			UselessFiles:    uselessFiles,
		})
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
