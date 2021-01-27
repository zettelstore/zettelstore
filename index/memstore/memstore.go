//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package memstore stored the index in main memory.
package memstore

import (
	"context"
	"sync"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/index"
)

type metaRefs struct {
	forward  []id.Zid
	backward []id.Zid
}

type zettelIndex struct {
	dead     string
	forward  []id.Zid
	backward []id.Zid
	meta     map[string]metaRefs
}

func (zi *zettelIndex) isEmpty() bool {
	if len(zi.forward) > 0 || len(zi.backward) > 0 || zi.dead != "" {
		return false
	}
	return zi.meta == nil || len(zi.meta) == 0
}

type memStore struct {
	mx  sync.RWMutex
	idx map[id.Zid]*zettelIndex

	// Stats
	updates uint64
}

// New returns a new memory-based index store.
func New() index.Store {
	return &memStore{
		idx: make(map[id.Zid]*zettelIndex),
	}
}

func (ms *memStore) Enrich(ctx context.Context, m *meta.Meta) {
	ms.mx.RLock()
	defer ms.mx.RUnlock()
	zi, ok := ms.idx[m.Zid]
	if !ok {
		return
	}
	var updated bool
	if zi.dead != "" {
		m.Set(meta.KeyDead, zi.dead)
		updated = true
	}
	back := zi.backward
	if len(zi.backward) > 0 {
		m.Set(meta.KeyBackward, refsToString(zi.backward))
		updated = true
	}
	if len(zi.forward) > 0 {
		m.Set(meta.KeyForward, refsToString(zi.forward))
		back = remRefs(back, zi.forward)
		updated = true
	}
	if len(zi.meta) > 0 {
		for k, refs := range zi.meta {
			if len(refs.backward) > 0 {
				m.Set(k, refsToString(refs.backward))
				back = remRefs(back, refs.backward)
				updated = true
			}
		}
	}
	if len(back) > 0 {
		m.Set(meta.KeyBack, refsToString(back))
		updated = true
	}
	if updated {
		ms.updates++
	}
}

func (ms *memStore) UpdateReferences(ctx context.Context, zidx *index.ZettelIndex) {
	ms.mx.Lock()
	defer ms.mx.Unlock()
	zi, ziExist := ms.idx[zidx.Zid]
	if !ziExist || zi == nil {
		zi = &zettelIndex{}
		ziExist = false
	}

	// Update dead references
	if drefs := zidx.GetDeadRefs(); len(drefs) > 0 {
		zi.dead = refsToString(drefs)
	} else {
		zi.dead = ""
	}

	// Update forward and backward references
	brefs := zidx.GetBackRefs()
	newRefs, remRefs := refsDiff(brefs, zi.forward)
	zi.forward = brefs
	for _, ref := range newRefs {
		bzi := ms.getEntry(ref)
		bzi.backward = addRef(bzi.backward, zidx.Zid)
	}
	for _, ref := range remRefs {
		bzi := ms.getEntry(ref)
		bzi.backward = remRef(bzi.backward, zidx.Zid)
	}

	// Update metadata references
	metarefs := zidx.GetMetaRefs()
	for key, mr := range zi.meta {
		if _, ok := metarefs[key]; ok {
			continue
		}
		ms.removeInverseMeta(zidx.Zid, key, mr.forward)
	}
	if len(metarefs) == 0 {
		zi.meta = nil
	} else {
		if zi.meta == nil {
			zi.meta = make(map[string]metaRefs)
		}
		for key, mrefs := range metarefs {
			mr := zi.meta[key]
			newRefs, remRefs := refsDiff(mrefs, mr.forward)
			mr.forward = mrefs
			zi.meta[key] = mr

			for _, ref := range newRefs {
				bzi := ms.getEntry(ref)
				if bzi.meta == nil {
					bzi.meta = make(map[string]metaRefs)
				}
				bmr := bzi.meta[key]
				bmr.backward = addRef(bmr.backward, zidx.Zid)
				bzi.meta[key] = bmr
			}
			ms.removeInverseMeta(zidx.Zid, key, remRefs)
		}
	}

	// Check if zi must be inserted into ms.idx
	if !ziExist && !zi.isEmpty() {
		ms.idx[zidx.Zid] = zi
	}
}

func (ms *memStore) getEntry(zid id.Zid) *zettelIndex {
	if zi, ok := ms.idx[zid]; ok {
		return zi
	}
	zi := &zettelIndex{}
	ms.idx[zid] = zi
	return zi
}

func (ms *memStore) DeleteZettel(ctx context.Context, zid id.Zid) {
	ms.mx.Lock()
	defer ms.mx.Unlock()

	zi, ok := ms.idx[zid]
	if !ok {
		return
	}

	for _, ref := range zi.forward {
		if fzi, ok := ms.idx[ref]; ok {
			fzi.backward = remRef(fzi.backward, zid)
		}
	}
	for _, ref := range zi.backward {
		if bzi, ok := ms.idx[ref]; ok {
			bzi.forward = remRef(bzi.forward, zid)
		}
	}
	if len(zi.meta) > 0 {
		for key, mrefs := range zi.meta {
			ms.removeInverseMeta(zid, key, mrefs.forward)
		}
	}
	delete(ms.idx, zid)
}

func (ms *memStore) removeInverseMeta(zid id.Zid, key string, forward []id.Zid) {
	// Must only be called if ms.mx is write-locked!
	for _, ref := range forward {
		if bzi, ok := ms.idx[ref]; ok {
			if bzi.meta != nil {
				if bmr, ok := bzi.meta[key]; ok {
					bmr.backward = remRef(bmr.backward, zid)
					if len(bmr.backward) > 0 || len(bmr.forward) > 0 {
						bzi.meta[key] = bmr
					} else {
						delete(bzi.meta, key)
						if len(bzi.meta) == 0 {
							bzi.meta = nil
						}
					}
				}
			}
		}
	}
}

func (ms *memStore) ReadStats(st *index.StoreStats) {
	ms.mx.RLock()
	st.Zettel = len(ms.idx)
	st.Updates = ms.updates
	ms.mx.RUnlock()
}
