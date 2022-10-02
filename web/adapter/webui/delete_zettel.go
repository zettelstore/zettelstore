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
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/c/maps"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/strfun"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/server"
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
		var incomingLinks simpleLinks
		if len(ms) > 1 {
			shadowedBox = ms[1].GetDefault(api.KeyBoxNumber, "???")
		} else {
			getTextTitle := wui.makeGetTextTitle(createGetMetadataFunc(ctx, getMeta), createEvalMetadataFunc(ctx, evaluate))
			incomingLinks = wui.encodeIncoming(m, getTextTitle)
		}
		uselessFiles := retrieveUselessFiles(m)

		user := server.GetUser(ctx)
		var base baseData
		wui.makeBaseData(ctx, wui.rtConfig.Get(ctx, m, api.KeyLang), "Delete Zettel "+m.Zid.String(), "", user, &base)
		wui.renderTemplate(ctx, w, id.DeleteTemplateZid, &base, struct {
			Zid             string
			MetaPairs       []meta.Pair
			HasShadows      bool
			ShadowedBox     string
			Incoming        simpleLinks
			HasUselessFiles bool
			UselessFiles    []string
		}{
			Zid:             zid.String(),
			MetaPairs:       m.ComputedPairs(),
			HasShadows:      shadowedBox != "",
			ShadowedBox:     shadowedBox,
			Incoming:        incomingLinks,
			HasUselessFiles: len(uselessFiles) > 0,
			UselessFiles:    uselessFiles,
		})
	}
}

func retrieveUselessFiles(m *meta.Meta) []string {
	if val, found := m.Get(api.KeyUselessFiles); found {
		return []string{val}
	}
	return nil
}

// MakePostDeleteZettelHandler creates a new HTTP handler to delete a zettel.
func (wui *WebUI) MakePostDeleteZettelHandler(deleteZettel *usecase.DeleteZettel) http.HandlerFunc {
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
		wui.redirectFound(w, r, wui.NewURLBuilder('/'))
	}
}

func (wui *WebUI) encodeIncoming(m *meta.Meta, getTextTitle getTextTitleFunc) simpleLinks {
	zidMap := make(strfun.Set)
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
				zidMap.Set(val)
			}
		case meta.TypeIDSet:
			addListValues(zidMap, m, inverseKey)
		}
	}
	return createSimpleLinks(wui.encodeZidLinks(maps.Keys(zidMap), getTextTitle))
}

func addListValues(zidMap strfun.Set, m *meta.Meta, key string) {
	if values, ok := m.GetList(key); ok {
		for _, val := range values {
			zidMap.Set(val)
		}
	}
}
