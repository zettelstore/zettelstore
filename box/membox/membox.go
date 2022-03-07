//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
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
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/logger"
	"zettelstore.de/z/search"
)

func init() {
	manager.Register(
		"mem",
		func(u *url.URL, cdata *manager.ConnectData) (box.ManagedBox, error) {
			return &memBox{
				log: kernel.Main.GetLogger(kernel.BoxService).Clone().
					Str("box", "mem").Int("boxnum", int64(cdata.Number)).Child(),
				u:         u,
				cdata:     *cdata,
				maxZettel: box.GetQueryInt(u, "max-zettel", 0, 127, 65535),
				maxBytes:  box.GetQueryInt(u, "max-bytes", 0, 65535, (1024*1024*1024)-1),
			}, nil
		})
}

type memBox struct {
	log       *logger.Logger
	u         *url.URL
	cdata     manager.ConnectData
	maxZettel int
	maxBytes  int
	mx        sync.RWMutex // Protects the following fields
	zettel    map[id.Zid]domain.Zettel
	curBytes  int
}

func (mb *memBox) notifyChanged(reason box.UpdateReason, zid id.Zid) {
	if chci := mb.cdata.Notify; chci != nil {
		chci <- box.UpdateInfo{Reason: reason, Zid: zid}
	}
}

func (mb *memBox) Location() string {
	return mb.u.String()
}

func (mb *memBox) Start(context.Context) error {
	mb.mx.Lock()
	mb.zettel = make(map[id.Zid]domain.Zettel)
	mb.curBytes = 0
	mb.mx.Unlock()
	mb.log.Trace().Int("max-zettel", int64(mb.maxZettel)).Int("max-bytes", int64(mb.maxBytes)).Msg("Start Box")
	return nil
}

func (mb *memBox) Stop(context.Context) {
	mb.mx.Lock()
	mb.zettel = nil
	mb.mx.Unlock()
}

func (mb *memBox) CanCreateZettel(context.Context) bool {
	mb.mx.RLock()
	defer mb.mx.RUnlock()
	return len(mb.zettel) < mb.maxZettel
}

func (mb *memBox) CreateZettel(_ context.Context, zettel domain.Zettel) (id.Zid, error) {
	mb.mx.Lock()
	newBytes := mb.curBytes + zettel.Length()
	if mb.maxZettel < len(mb.zettel) || mb.maxBytes < newBytes {
		mb.mx.Unlock()
		return id.Invalid, box.ErrCapacity
	}
	zid, err := box.GetNewZid(func(zid id.Zid) (bool, error) {
		_, ok := mb.zettel[zid]
		return !ok, nil
	})
	if err != nil {
		mb.mx.Unlock()
		return id.Invalid, err
	}
	meta := zettel.Meta.Clone()
	meta.Zid = zid
	zettel.Meta = meta
	mb.zettel[zid] = zettel
	mb.curBytes = newBytes
	mb.mx.Unlock()
	mb.notifyChanged(box.OnUpdate, zid)
	mb.log.Trace().Zid(zid).Msg("CreateZettel")
	return zid, nil
}

func (mb *memBox) GetZettel(_ context.Context, zid id.Zid) (domain.Zettel, error) {
	mb.mx.RLock()
	zettel, ok := mb.zettel[zid]
	mb.mx.RUnlock()
	if !ok {
		return domain.Zettel{}, box.ErrNotFound
	}
	zettel.Meta = zettel.Meta.Clone()
	mb.log.Trace().Msg("GetZettel")
	return zettel, nil
}

func (mb *memBox) GetMeta(_ context.Context, zid id.Zid) (*meta.Meta, error) {
	mb.mx.RLock()
	zettel, ok := mb.zettel[zid]
	mb.mx.RUnlock()
	if !ok {
		return nil, box.ErrNotFound
	}
	mb.log.Trace().Msg("GetMeta")
	return zettel.Meta.Clone(), nil
}

func (mb *memBox) ApplyZid(_ context.Context, handle box.ZidFunc, constraint search.RetrievePredicate) error {
	mb.mx.RLock()
	defer mb.mx.RUnlock()
	mb.log.Trace().Int("entries", int64(len(mb.zettel))).Msg("ApplyZid")
	for zid := range mb.zettel {
		if constraint(zid) {
			handle(zid)
		}
	}
	return nil
}

func (mb *memBox) ApplyMeta(ctx context.Context, handle box.MetaFunc, constraint search.RetrievePredicate) error {
	mb.mx.RLock()
	defer mb.mx.RUnlock()
	mb.log.Trace().Int("entries", int64(len(mb.zettel))).Msg("ApplyMeta")
	for zid, zettel := range mb.zettel {
		if constraint(zid) {
			m := zettel.Meta.Clone()
			mb.cdata.Enricher.Enrich(ctx, m, mb.cdata.Number)
			handle(m)
		}
	}
	return nil
}

func (mb *memBox) CanUpdateZettel(_ context.Context, zettel domain.Zettel) bool {
	mb.mx.RLock()
	defer mb.mx.RUnlock()
	zid := zettel.Meta.Zid
	if !zid.IsValid() {
		return false
	}

	newBytes := mb.curBytes + zettel.Length()
	if prevZettel, found := mb.zettel[zid]; found {
		newBytes -= prevZettel.Length()
	}
	return newBytes < mb.maxBytes
}

func (mb *memBox) UpdateZettel(_ context.Context, zettel domain.Zettel) error {
	m := zettel.Meta.Clone()
	if !m.Zid.IsValid() {
		return &box.ErrInvalidID{Zid: m.Zid}
	}

	mb.mx.Lock()
	newBytes := mb.curBytes + zettel.Length()
	if prevZettel, found := mb.zettel[m.Zid]; found {
		newBytes -= prevZettel.Length()
	}
	if mb.maxBytes < newBytes {
		mb.mx.Unlock()
		return box.ErrCapacity
	}

	zettel.Meta = m
	mb.zettel[m.Zid] = zettel
	mb.curBytes = newBytes
	mb.mx.Unlock()
	mb.notifyChanged(box.OnUpdate, m.Zid)
	mb.log.Trace().Msg("UpdateZettel")
	return nil
}

func (*memBox) AllowRenameZettel(context.Context, id.Zid) bool { return true }

func (mb *memBox) RenameZettel(_ context.Context, curZid, newZid id.Zid) error {
	mb.mx.Lock()
	zettel, ok := mb.zettel[curZid]
	if !ok {
		mb.mx.Unlock()
		return box.ErrNotFound
	}

	// Check that there is no zettel with newZid
	if _, ok = mb.zettel[newZid]; ok {
		mb.mx.Unlock()
		return &box.ErrInvalidID{Zid: newZid}
	}

	meta := zettel.Meta.Clone()
	meta.Zid = newZid
	zettel.Meta = meta
	mb.zettel[newZid] = zettel
	delete(mb.zettel, curZid)
	mb.mx.Unlock()
	mb.notifyChanged(box.OnDelete, curZid)
	mb.notifyChanged(box.OnUpdate, newZid)
	mb.log.Trace().Msg("RenameZettel")
	return nil
}

func (mb *memBox) CanDeleteZettel(_ context.Context, zid id.Zid) bool {
	mb.mx.RLock()
	_, ok := mb.zettel[zid]
	mb.mx.RUnlock()
	return ok
}

func (mb *memBox) DeleteZettel(_ context.Context, zid id.Zid) error {
	mb.mx.Lock()
	oldZettel, found := mb.zettel[zid]
	if !found {
		mb.mx.Unlock()
		return box.ErrNotFound
	}
	delete(mb.zettel, zid)
	mb.curBytes -= oldZettel.Length()
	mb.mx.Unlock()
	mb.notifyChanged(box.OnDelete, zid)
	mb.log.Trace().Msg("DeleteZettel")
	return nil
}

func (mb *memBox) ReadStats(st *box.ManagedBoxStats) {
	st.ReadOnly = false
	mb.mx.RLock()
	st.Zettel = len(mb.zettel)
	mb.mx.RUnlock()
	mb.log.Trace().Int("zettel", int64(st.Zettel)).Msg("ReadStats")
}
