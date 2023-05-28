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
	"net/http"

	"codeberg.org/t73fde/sxpf"
	"zettelstore.de/c/api"
	"zettelstore.de/c/maps"
	"zettelstore.de/z/box"
	"zettelstore.de/z/strfun"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/server"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

// MakeGetDeleteZettelHandler creates a new HTTP handler to display the
// HTML delete view of a zettel.
func (wui *WebUI) MakeGetDeleteZettelHandler(getMeta usecase.GetMeta, getAllMeta usecase.GetAllMeta) http.HandlerFunc {
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

		user := server.GetUser(ctx)
		env, err := wui.createRenderEnv(
			ctx, "delete",
			wui.rtConfig.Get(ctx, nil, api.KeyLang), "Delete Zettel "+m.Zid.String(), user)
		rb := makeRenderBinder(wui.sf, env, err)
		rb.bindString("zid", sxpf.MakeString(m.Zid.String()))
		if len(ms) > 1 {
			rb.bindString("shadowed-box", sxpf.MakeString(ms[1].GetDefault(api.KeyBoxNumber, "???")))
			rb.bindString("incoming", nil)
		} else {
			rb.bindString("shadowed-box", nil)
			rb.bindString("incoming", wui.encodeIncoming(m, wui.makeGetTextTitle(ctx, getMeta)))
		}
		rb.bindString("useless", retrieveUselessFiles(m))
		rb.bindString("meta-pairs", makeMetaPairs(m))
		if rb.err == nil {
			err = wui.renderSxnTemplate(ctx, w, id.DeleteTemplateZid, env)
		}
		if err != nil {
			wui.reportError(ctx, w, err)
		}
	}
}

func retrieveUselessFiles(m *meta.Meta) *sxpf.List {
	if val, found := m.Get(api.KeyUselessFiles); found {
		return sxpf.Cons(sxpf.MakeString(val), nil)
	}
	return nil
}

func (wui *WebUI) encodeIncoming(m *meta.Meta, getTextTitle getTextTitleFunc) *sxpf.List {
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
	return wui.zidLinksSxn(maps.Keys(zidMap), getTextTitle)
}

func addListValues(zidMap strfun.Set, m *meta.Meta, key string) {
	if values, ok := m.GetList(key); ok {
		for _, val := range values {
			zidMap.Set(val)
		}
	}
}

func makeMetaPairs(m *meta.Meta) *sxpf.List {
	sentinel := sxpf.Cons(nil, nil)
	curr := sentinel
	for _, p := range m.ComputedPairs() {
		curr = curr.AppendBang(sxpf.Cons(sxpf.MakeString(p.Key), sxpf.MakeString(p.Value)))
	}
	return sentinel.Tail()
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
