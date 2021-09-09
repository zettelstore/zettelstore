//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package membox stores zettel volatile in main memory.
package membox

import (
	"context"
	"net/url"
	"sync"

	"zettelstore.de/z/box"
	"zettelstore.de/z/box/manager"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

func init() {
	manager.Register(
		"mem",
		func(u *url.URL, cdata *manager.ConnectData) (box.ManagedBox, error) {
			return &memBox{u: u, cdata: *cdata}, nil
		})
}

type memBox struct {
	u      *url.URL
	cdata  manager.ConnectData
	zettel map[id.Zid]domain.Zettel
	mx     sync.RWMutex
}

func (mp *memBox) notifyChanged(reason box.UpdateReason, zid id.Zid) {
	if chci := mp.cdata.Notify; chci != nil {
		chci <- box.UpdateInfo{Reason: reason, Zid: zid}
	}
}

func (mp *memBox) Location() string {
	return mp.u.String()
}

func (mp *memBox) Start(context.Context) error {
	mp.mx.Lock()
	mp.zettel = make(map[id.Zid]domain.Zettel)
	mp.mx.Unlock()
	return nil
}

func (mp *memBox) Stop(context.Context) error {
	mp.mx.Lock()
	mp.zettel = nil
	mp.mx.Unlock()
	return nil
}

func (mp *memBox) CanCreateZettel(context.Context) bool { return true }

func (mp *memBox) CreateZettel(_ context.Context, zettel domain.Zettel) (id.Zid, error) {
	mp.mx.Lock()
	zid, err := box.GetNewZid(func(zid id.Zid) (bool, error) {
		_, ok := mp.zettel[zid]
		return !ok, nil
	})
	if err != nil {
		mp.mx.Unlock()
		return id.Invalid, err
	}
	meta := zettel.Meta.Clone()
	meta.Zid = zid
	zettel.Meta = meta
	mp.zettel[zid] = zettel
	mp.mx.Unlock()
	mp.notifyChanged(box.OnUpdate, zid)
	return zid, nil
}

func (mp *memBox) GetZettel(_ context.Context, zid id.Zid) (domain.Zettel, error) {
	mp.mx.RLock()
	zettel, ok := mp.zettel[zid]
	mp.mx.RUnlock()
	if !ok {
		return domain.Zettel{}, box.ErrNotFound
	}
	zettel.Meta = zettel.Meta.Clone()
	return zettel, nil
}

func (mp *memBox) GetMeta(_ context.Context, zid id.Zid) (*meta.Meta, error) {
	mp.mx.RLock()
	zettel, ok := mp.zettel[zid]
	mp.mx.RUnlock()
	if !ok {
		return nil, box.ErrNotFound
	}
	return zettel.Meta.Clone(), nil
}

func (mp *memBox) ApplyZid(_ context.Context, handle box.ZidFunc) error {
	mp.mx.RLock()
	defer mp.mx.RUnlock()
	for zid := range mp.zettel {
		handle(zid)
	}
	return nil
}

func (mp *memBox) ApplyMeta(ctx context.Context, handle box.MetaFunc) error {
	mp.mx.RLock()
	defer mp.mx.RUnlock()
	for _, zettel := range mp.zettel {
		m := zettel.Meta.Clone()
		mp.cdata.Enricher.Enrich(ctx, m, mp.cdata.Number)
		handle(m)
	}
	return nil
}

func (mp *memBox) CanUpdateZettel(context.Context, domain.Zettel) bool {
	return true
}

func (mp *memBox) UpdateZettel(_ context.Context, zettel domain.Zettel) error {
	mp.mx.Lock()
	meta := zettel.Meta.Clone()
	if !meta.Zid.IsValid() {
		return &box.ErrInvalidID{Zid: meta.Zid}
	}
	zettel.Meta = meta
	mp.zettel[meta.Zid] = zettel
	mp.mx.Unlock()
	mp.notifyChanged(box.OnUpdate, meta.Zid)
	return nil
}

func (mp *memBox) AllowRenameZettel(context.Context, id.Zid) bool { return true }

func (mp *memBox) RenameZettel(_ context.Context, curZid, newZid id.Zid) error {
	mp.mx.Lock()
	zettel, ok := mp.zettel[curZid]
	if !ok {
		mp.mx.Unlock()
		return box.ErrNotFound
	}

	// Check that there is no zettel with newZid
	if _, ok = mp.zettel[newZid]; ok {
		mp.mx.Unlock()
		return &box.ErrInvalidID{Zid: newZid}
	}

	meta := zettel.Meta.Clone()
	meta.Zid = newZid
	zettel.Meta = meta
	mp.zettel[newZid] = zettel
	delete(mp.zettel, curZid)
	mp.mx.Unlock()
	mp.notifyChanged(box.OnDelete, curZid)
	mp.notifyChanged(box.OnUpdate, newZid)
	return nil
}

func (mp *memBox) CanDeleteZettel(_ context.Context, zid id.Zid) bool {
	mp.mx.RLock()
	_, ok := mp.zettel[zid]
	mp.mx.RUnlock()
	return ok
}

func (mp *memBox) DeleteZettel(_ context.Context, zid id.Zid) error {
	mp.mx.Lock()
	if _, ok := mp.zettel[zid]; !ok {
		mp.mx.Unlock()
		return box.ErrNotFound
	}
	delete(mp.zettel, zid)
	mp.mx.Unlock()
	mp.notifyChanged(box.OnDelete, zid)
	return nil
}

func (mp *memBox) ReadStats(st *box.ManagedBoxStats) {
	st.ReadOnly = false
	mp.mx.RLock()
	st.Zettel = len(mp.zettel)
	mp.mx.RUnlock()
}
