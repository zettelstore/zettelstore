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

type zettelIndex struct {
	dead     string
	forward  []id.Zid
	backward []id.Zid
}

type memStore struct {
	mx  sync.RWMutex
	idx map[id.Zid]*zettelIndex
}

// New returns a new memory-based index store.
func New() index.Store {
	return &memStore{
		idx: make(map[id.Zid]*zettelIndex),
	}
}

func (ms *memStore) Update(ctx context.Context, m *meta.Meta) {
	ms.mx.RLock()
	defer ms.mx.RUnlock()
	zi, ok := ms.idx[m.Zid]
	if !ok {
		return
	}
	if zi.dead != "" {
		m.Set(meta.KeyDead, zi.dead)
	}
	if len(zi.backward) > 0 {
		m.Set(meta.KeyBack, refsToString(zi.backward))
	}
}

func (ms *memStore) UpdateReferences(ctx context.Context, zidx *index.ZettelIndex) {
	ms.mx.Lock()
	defer ms.mx.Unlock()
	zi := ms.getEntry(zidx.Zid)

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

	// Too simple... needs elaboration
	delete(ms.idx, zid)
}
