//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package memplace stores zettel volatile in main memory.
package memplace

import (
	"context"
	"net/url"
	"sync"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/change"
	"zettelstore.de/z/place/manager"
	"zettelstore.de/z/search"
)

func init() {
	manager.Register(
		"mem",
		func(u *url.URL, cdata *manager.ConnectData) (place.ManagedPlace, error) {
			return &memPlace{u: u, cdata: *cdata}, nil
		})
}

type memPlace struct {
	u      *url.URL
	cdata  manager.ConnectData
	zettel map[id.Zid]domain.Zettel
	mx     sync.RWMutex
}

func (mp *memPlace) notifyChanged(reason change.Reason, zid id.Zid) {
	if chci := mp.cdata.Notify; chci != nil {
		chci <- change.Info{Reason: reason, Zid: zid}
	}
}

func (mp *memPlace) Location() string {
	return mp.u.String()
}

func (mp *memPlace) Start(ctx context.Context) error {
	mp.mx.Lock()
	mp.zettel = make(map[id.Zid]domain.Zettel)
	mp.mx.Unlock()
	return nil
}

func (mp *memPlace) Stop(ctx context.Context) error {
	mp.mx.Lock()
	mp.zettel = nil
	mp.mx.Unlock()
	return nil
}

func (mp *memPlace) CanCreateZettel(ctx context.Context) bool { return true }

func (mp *memPlace) CreateZettel(ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	mp.mx.Lock()
	zid, err := place.GetNewZid(func(zid id.Zid) (bool, error) {
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
	mp.notifyChanged(change.OnUpdate, zid)
	return zid, nil
}

func (mp *memPlace) GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error) {
	mp.mx.RLock()
	zettel, ok := mp.zettel[zid]
	mp.mx.RUnlock()
	if !ok {
		return domain.Zettel{}, place.ErrNotFound
	}
	zettel.Meta = zettel.Meta.Clone()
	return zettel, nil
}

func (mp *memPlace) GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	mp.mx.RLock()
	zettel, ok := mp.zettel[zid]
	mp.mx.RUnlock()
	if !ok {
		return nil, place.ErrNotFound
	}
	return zettel.Meta.Clone(), nil
}

func (mp *memPlace) FetchZids(ctx context.Context) (id.Set, error) {
	mp.mx.RLock()
	result := id.NewSetCap(len(mp.zettel))
	for zid := range mp.zettel {
		result[zid] = true
	}
	mp.mx.RUnlock()
	return result, nil
}

func (mp *memPlace) SelectMeta(ctx context.Context, match search.MetaMatchFunc) ([]*meta.Meta, error) {
	result := make([]*meta.Meta, 0, len(mp.zettel))
	mp.mx.RLock()
	for _, zettel := range mp.zettel {
		m := zettel.Meta.Clone()
		mp.cdata.Enricher.Enrich(ctx, m)
		if match(m) {
			result = append(result, m)
		}
	}
	mp.mx.RUnlock()
	return result, nil
}

func (mp *memPlace) CanUpdateZettel(ctx context.Context, zettel domain.Zettel) bool {
	return true
}

func (mp *memPlace) UpdateZettel(ctx context.Context, zettel domain.Zettel) error {
	mp.mx.Lock()
	meta := zettel.Meta.Clone()
	if !meta.Zid.IsValid() {
		return &place.ErrInvalidID{Zid: meta.Zid}
	}
	zettel.Meta = meta
	mp.zettel[meta.Zid] = zettel
	mp.mx.Unlock()
	mp.notifyChanged(change.OnUpdate, meta.Zid)
	return nil
}

func (mp *memPlace) AllowRenameZettel(ctx context.Context, zid id.Zid) bool { return true }

func (mp *memPlace) RenameZettel(ctx context.Context, curZid, newZid id.Zid) error {
	mp.mx.Lock()
	zettel, ok := mp.zettel[curZid]
	if !ok {
		mp.mx.Unlock()
		return place.ErrNotFound
	}

	// Check that there is no zettel with newZid
	if _, ok = mp.zettel[newZid]; ok {
		mp.mx.Unlock()
		return &place.ErrInvalidID{Zid: newZid}
	}

	meta := zettel.Meta.Clone()
	meta.Zid = newZid
	zettel.Meta = meta
	mp.zettel[newZid] = zettel
	delete(mp.zettel, curZid)
	mp.mx.Unlock()
	mp.notifyChanged(change.OnDelete, curZid)
	mp.notifyChanged(change.OnUpdate, newZid)
	return nil
}

func (mp *memPlace) CanDeleteZettel(ctx context.Context, zid id.Zid) bool {
	mp.mx.RLock()
	_, ok := mp.zettel[zid]
	mp.mx.RUnlock()
	return ok
}

func (mp *memPlace) DeleteZettel(ctx context.Context, zid id.Zid) error {
	mp.mx.Lock()
	if _, ok := mp.zettel[zid]; !ok {
		mp.mx.Unlock()
		return place.ErrNotFound
	}
	delete(mp.zettel, zid)
	mp.mx.Unlock()
	mp.notifyChanged(change.OnDelete, zid)
	return nil
}

func (mp *memPlace) ReadStats(st *place.Stats) {
	st.ReadOnly = false
	mp.mx.RLock()
	st.Zettel = len(mp.zettel)
	mp.mx.RUnlock()
}
