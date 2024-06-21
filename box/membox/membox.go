//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2020-present Detlef Stern
//-----------------------------------------------------------------------------

// Package membox stores zettel volatile in main memory.
package membox

import (
	"context"
	"net/url"
	"sync"

	"zettelstore.de/z/box"
	"zettelstore.de/z/box/manager"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/logger"
	"zettelstore.de/z/query"
	"zettelstore.de/z/zettel"
	"zettelstore.de/z/zettel/id"
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
	zettel    map[id.ZidO]zettel.Zettel
	curBytes  int
}

func (mb *memBox) notifyChanged(zid id.ZidO) {
	if chci := mb.cdata.Notify; chci != nil {
		chci <- box.UpdateInfo{Box: mb, Reason: box.OnZettel, Zid: zid}
	}
}

func (mb *memBox) Location() string {
	return mb.u.String()
}

func (mb *memBox) State() box.StartState {
	mb.mx.RLock()
	defer mb.mx.RUnlock()
	if mb.zettel == nil {
		return box.StartStateStopped
	}
	return box.StartStateStarted
}

func (mb *memBox) Start(context.Context) error {
	mb.mx.Lock()
	mb.zettel = make(map[id.ZidO]zettel.Zettel)
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

func (mb *memBox) CreateZettel(_ context.Context, zettel zettel.Zettel) (id.ZidO, error) {
	mb.mx.Lock()
	newBytes := mb.curBytes + zettel.Length()
	if mb.maxZettel < len(mb.zettel) || mb.maxBytes < newBytes {
		mb.mx.Unlock()
		return id.InvalidO, box.ErrCapacity
	}
	zid, err := box.GetNewZid(func(zid id.ZidO) (bool, error) {
		_, ok := mb.zettel[zid]
		return !ok, nil
	})
	if err != nil {
		mb.mx.Unlock()
		return id.InvalidO, err
	}
	meta := zettel.Meta.Clone()
	meta.ZidO = zid
	zettel.Meta = meta
	mb.zettel[zid] = zettel
	mb.curBytes = newBytes
	mb.mx.Unlock()
	mb.notifyChanged(zid)
	mb.log.Trace().Zid(zid).Msg("CreateZettel")
	return zid, nil
}

func (mb *memBox) GetZettel(_ context.Context, zid id.ZidO) (zettel.Zettel, error) {
	mb.mx.RLock()
	z, ok := mb.zettel[zid]
	mb.mx.RUnlock()
	if !ok {
		return zettel.Zettel{}, box.ErrZettelNotFound{Zid: zid}
	}
	z.Meta = z.Meta.Clone()
	mb.log.Trace().Msg("GetZettel")
	return z, nil
}

func (mb *memBox) HasZettel(_ context.Context, zid id.ZidO) bool {
	mb.mx.RLock()
	_, found := mb.zettel[zid]
	mb.mx.RUnlock()
	return found
}

func (mb *memBox) ApplyZid(_ context.Context, handle box.ZidFunc, constraint query.RetrievePredicate) error {
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

func (mb *memBox) ApplyMeta(ctx context.Context, handle box.MetaFunc, constraint query.RetrievePredicate) error {
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

func (mb *memBox) CanUpdateZettel(_ context.Context, zettel zettel.Zettel) bool {
	mb.mx.RLock()
	defer mb.mx.RUnlock()
	zid := zettel.Meta.ZidO
	if !zid.IsValid() {
		return false
	}

	newBytes := mb.curBytes + zettel.Length()
	if prevZettel, found := mb.zettel[zid]; found {
		newBytes -= prevZettel.Length()
	}
	return newBytes < mb.maxBytes
}

func (mb *memBox) UpdateZettel(_ context.Context, zettel zettel.Zettel) error {
	m := zettel.Meta.Clone()
	if !m.ZidO.IsValid() {
		return box.ErrInvalidZid{Zid: m.ZidO.String()}
	}

	mb.mx.Lock()
	newBytes := mb.curBytes + zettel.Length()
	if prevZettel, found := mb.zettel[m.ZidO]; found {
		newBytes -= prevZettel.Length()
	}
	if mb.maxBytes < newBytes {
		mb.mx.Unlock()
		return box.ErrCapacity
	}

	zettel.Meta = m
	mb.zettel[m.ZidO] = zettel
	mb.curBytes = newBytes
	mb.mx.Unlock()
	mb.notifyChanged(m.ZidO)
	mb.log.Trace().Msg("UpdateZettel")
	return nil
}

func (*memBox) AllowRenameZettel(context.Context, id.ZidO) bool { return true }

func (mb *memBox) RenameZettel(_ context.Context, curZid, newZid id.ZidO) error {
	mb.mx.Lock()
	zettel, ok := mb.zettel[curZid]
	if !ok {
		mb.mx.Unlock()
		return box.ErrZettelNotFound{Zid: curZid}
	}

	// Check that there is no zettel with newZid
	if _, ok = mb.zettel[newZid]; ok {
		mb.mx.Unlock()
		return box.ErrInvalidZid{Zid: newZid.String()}
	}

	meta := zettel.Meta.Clone()
	meta.ZidO = newZid
	zettel.Meta = meta
	mb.zettel[newZid] = zettel
	delete(mb.zettel, curZid)
	mb.mx.Unlock()
	mb.notifyChanged(curZid)
	mb.notifyChanged(newZid)
	mb.log.Trace().Msg("RenameZettel")
	return nil
}

func (mb *memBox) CanDeleteZettel(_ context.Context, zid id.ZidO) bool {
	mb.mx.RLock()
	_, ok := mb.zettel[zid]
	mb.mx.RUnlock()
	return ok
}

func (mb *memBox) DeleteZettel(_ context.Context, zid id.ZidO) error {
	mb.mx.Lock()
	oldZettel, found := mb.zettel[zid]
	if !found {
		mb.mx.Unlock()
		return box.ErrZettelNotFound{Zid: zid}
	}
	delete(mb.zettel, zid)
	mb.curBytes -= oldZettel.Length()
	mb.mx.Unlock()
	mb.notifyChanged(zid)
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
