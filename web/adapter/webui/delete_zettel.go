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
	"fmt"
	"net/http"
	"sort"

	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetDeleteZettelHandler creates a new HTTP handler to display the
// HTML delete view of a zettel.
func (wui *WebUI) MakeGetDeleteZettelHandler(
	getMeta usecase.GetMeta,
	getAllMeta usecase.GetAllMeta,
	evaluate *usecase.Evaluate,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if enc, encText := adapter.GetEncoding(r, r.URL.Query(), api.EncoderHTML); enc != api.EncoderHTML {
			wui.reportError(ctx, w, adapter.NewErrBadRequest(
				fmt.Sprintf("Delete zettel not possible in encoding %q", encText)))
			return
		}

		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			wui.reportError(ctx, w, box.ErrNotFound)
			return
		}

		ms, err := getAllMeta.Run(ctx, zid)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		m := ms[0]

		var shadowedBox string
		var incomingLinks []simpleLink
		if len(ms) > 1 {
			shadowedBox = ms[1].GetDefault(api.KeyBoxNumber, "???")
		} else {
			getTextTitle := wui.makeGetTextTitle(ctx, getMeta, evaluate)
			incomingLinks = wui.encodeIncoming(m, getTextTitle)
		}

		user := wui.getUser(ctx)
		var base baseData
		wui.makeBaseData(ctx, config.GetLang(m, wui.rtConfig), "Delete Zettel "+m.Zid.String(), user, &base)
		wui.renderTemplate(ctx, w, id.DeleteTemplateZid, &base, struct {
			Zid         string
			MetaPairs   []meta.Pair
			HasShadows  bool
			ShadowedBox string
			HasIncoming bool
			Incoming    []simpleLink
		}{
			Zid:         zid.String(),
			MetaPairs:   m.Pairs(true),
			HasShadows:  shadowedBox != "",
			ShadowedBox: shadowedBox,
			HasIncoming: len(incomingLinks) > 0,
			Incoming:    incomingLinks,
		})
	}
}

// MakePostDeleteZettelHandler creates a new HTTP handler to delete a zettel.
func (wui *WebUI) MakePostDeleteZettelHandler(deleteZettel usecase.DeleteZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			wui.reportError(ctx, w, box.ErrNotFound)
			return
		}

		if err = deleteZettel.Run(r.Context(), zid); err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		redirectFound(w, r, wui.NewURLBuilder('/'))
	}
}

func (wui *WebUI) encodeIncoming(m *meta.Meta, getTextTitle getTextTitleFunc) []simpleLink {
	zidMap := make(map[string]bool)
	addListValues(zidMap, m, api.KeyBackward)
	for _, kd := range meta.GetSortedKeyDescriptions() {
		inverseKey := kd.Inverse
		if inverseKey == "" {
			continue
		}
		ikd := meta.GetDescription(inverseKey)
		switch ikd.Type {
		case meta.TypeID:
			if val, ok := m.Get(inverseKey); ok {
				zidMap[val] = true
			}
		case meta.TypeIDSet:
			addListValues(zidMap, m, inverseKey)
		}
	}
	values := make([]string, 0, len(zidMap))
	for val := range zidMap {
		values = append(values, val)
	}
	sort.Strings(values)
	return wui.encodeZidLinks(values, getTextTitle)
}

func addListValues(zidMap map[string]bool, m *meta.Meta, key string) {
	if values, ok := m.GetList(key); ok {
		for _, val := range values {
			zidMap[val] = true
		}
	}
}
