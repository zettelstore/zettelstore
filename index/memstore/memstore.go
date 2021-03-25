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
	"fmt"
	"io"
	"strings"
	"sync"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/index"
)

type metaRefs struct {
	forward  id.Slice
	backward id.Slice
}

type zettelIndex struct {
	dead     id.Slice
	forward  id.Slice
	backward id.Slice
	meta     map[string]metaRefs
}

func (zi *zettelIndex) isEmpty() bool {
	if len(zi.forward) > 0 || len(zi.backward) > 0 || len(zi.dead) > 0 {
		return false
	}
	return zi.meta == nil || len(zi.meta) == 0
}

type memStore struct {
	mx    sync.RWMutex
	idx   map[id.Zid]*zettelIndex
	dead  map[id.Zid]id.Slice // map dead refs where they occur
	words map[string]id.Slice

	// Stats
	updates uint64
}

// New returns a new memory-based index store.
func New() index.Store {
	return &memStore{
		idx:   make(map[id.Zid]*zettelIndex),
		dead:  make(map[id.Zid]id.Slice),
		words: make(map[string]id.Slice),
	}
}

func (ms *memStore) Enrich(ctx context.Context, m *meta.Meta) {
	if ms.doEnrich(ctx, m) {
		ms.mx.Lock()
		ms.updates++
		ms.mx.Unlock()
	}
}

func (ms *memStore) doEnrich(ctx context.Context, m *meta.Meta) bool {
	ms.mx.RLock()
	defer ms.mx.RUnlock()
	zi, ok := ms.idx[m.Zid]
	if !ok {
		return false
	}
	var updated bool
	if len(zi.dead) > 0 {
		m.Set(meta.KeyDead, zi.dead.String())
		updated = true
	}
	back := removeOtherMetaRefs(m, zi.backward.Copy())
	if len(zi.backward) > 0 {
		m.Set(meta.KeyBackward, zi.backward.String())
		updated = true
	}
	if len(zi.forward) > 0 {
		m.Set(meta.KeyForward, zi.forward.String())
		back = remRefs(back, zi.forward)
		updated = true
	}
	if len(zi.meta) > 0 {
		for k, refs := range zi.meta {
			if len(refs.backward) > 0 {
				m.Set(k, refs.backward.String())
				back = remRefs(back, refs.backward)
				updated = true
			}
		}
	}
	if len(back) > 0 {
		m.Set(meta.KeyBack, back.String())
		updated = true
	}
	return updated
}

// Select all zettel that contains the given exact word.
// The word must be normalized through Unicode NKFD.
func (ms *memStore) Select(word string) id.Set {
	ms.mx.RLock()
	defer ms.mx.RUnlock()
	if refs, ok := ms.words[word]; ok {
		return id.NewSet(refs...)
	}
	return nil
}

// Select all zettel that have a word with the given prefix.
// The prefix must be normalized through Unicode NKFD.
func (ms *memStore) SelectPrefix(prefix string) id.Set {
	ms.mx.RLock()
	defer ms.mx.RUnlock()
	return ms.selectWithPred(prefix, strings.HasPrefix)
}

// Select all zettel that contains the given string.
// The string must be normalized through Unicode NKFD.
func (ms *memStore) SelectContains(s string) id.Set {
	ms.mx.RLock()
	defer ms.mx.RUnlock()
	return ms.selectWithPred(s, strings.Contains)
}

func (ms *memStore) selectWithPred(s string, pred func(string, string) bool) id.Set {
	result := id.NewSet()
	for word, refs := range ms.words {
		if !pred(word, s) {
			continue
		}
		for _, ref := range refs {
			result[ref] = true
		}
	}
	return result
}

func removeOtherMetaRefs(m *meta.Meta, back id.Slice) id.Slice {
	for _, p := range m.PairsRest(false) {
		switch meta.Type(p.Key) {
		case meta.TypeID:
			if zid, err := id.Parse(p.Value); err == nil {
				back = remRef(back, zid)
			}
		case meta.TypeIDSet:
			for _, val := range meta.ListFromValue(p.Value) {
				if zid, err := id.Parse(val); err == nil {
					back = remRef(back, zid)
				}
			}
		}
	}
	return back
}

func (ms *memStore) UpdateReferences(ctx context.Context, zidx *index.ZettelIndex) id.Set {
	ms.mx.Lock()
	defer ms.mx.Unlock()
	zi, ziExist := ms.idx[zidx.Zid]
	if !ziExist || zi == nil {
		zi = &zettelIndex{}
		ziExist = false
	}

	// Is this zettel an old dead reference mentioned in other zettel?
	var toCheck id.Set
	if refs, ok := ms.dead[zidx.Zid]; ok {
		// These must be checked later again
		toCheck = id.NewSet(refs...)
		delete(ms.dead, zidx.Zid)
	}

	ms.updateDeadReferences(zidx, zi)
	ms.updateForwardBackwardReferences(zidx, zi)
	ms.updateMetadataReferences(zidx, zi)
	ms.updateWords(zidx)

	// Check if zi must be inserted into ms.idx
	if !ziExist && !zi.isEmpty() {
		ms.idx[zidx.Zid] = zi
	}

	return toCheck
}

func (ms *memStore) updateDeadReferences(zidx *index.ZettelIndex, zi *zettelIndex) {
	// Must only be called if ms.mx is write-locked!
	drefs := zidx.GetDeadRefs()
	newRefs, remRefs := refsDiff(drefs, zi.dead)
	zi.dead = drefs
	for _, ref := range remRefs {
		ms.dead[ref] = remRef(ms.dead[ref], zidx.Zid)
	}
	for _, ref := range newRefs {
		ms.dead[ref] = addRef(ms.dead[ref], zidx.Zid)
	}
}

func (ms *memStore) updateForwardBackwardReferences(zidx *index.ZettelIndex, zi *zettelIndex) {
	// Must only be called if ms.mx is write-locked!
	brefs := zidx.GetBackRefs()
	newRefs, remRefs := refsDiff(brefs, zi.forward)
	zi.forward = brefs
	for _, ref := range remRefs {
		bzi := ms.getEntry(ref)
		bzi.backward = remRef(bzi.backward, zidx.Zid)
	}
	for _, ref := range newRefs {
		bzi := ms.getEntry(ref)
		bzi.backward = addRef(bzi.backward, zidx.Zid)
	}
}

func (ms *memStore) updateMetadataReferences(zidx *index.ZettelIndex, zi *zettelIndex) {
	// Must only be called if ms.mx is write-locked!
	metarefs := zidx.GetMetaRefs()
	for key, mr := range zi.meta {
		if _, ok := metarefs[key]; ok {
			continue
		}
		ms.removeInverseMeta(zidx.Zid, key, mr.forward)
	}
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

func (ms *memStore) updateWords(zidx *index.ZettelIndex) {
	// Must only be called if ms.mx is write-locked!
	words := zidx.GetWords()
	for word := range words {
		if refs, ok := ms.words[word]; ok {
			ms.words[word] = addRef(refs, zidx.Zid)
			continue
		}
		ms.words[word] = id.Slice{zidx.Zid}
	}
}

func (ms *memStore) getEntry(zid id.Zid) *zettelIndex {
	// Must only be called if ms.mx is write-locked!
	if zi, ok := ms.idx[zid]; ok {
		return zi
	}
	zi := &zettelIndex{}
	ms.idx[zid] = zi
	return zi
}

func (ms *memStore) DeleteZettel(ctx context.Context, zid id.Zid) id.Set {
	ms.mx.Lock()
	defer ms.mx.Unlock()

	zi, ok := ms.idx[zid]
	if !ok {
		return nil
	}

	ms.deleteDeadSources(zid, zi)
	toCheck := ms.deleteForwardBackward(zid, zi)
	if len(zi.meta) > 0 {
		for key, mrefs := range zi.meta {
			ms.removeInverseMeta(zid, key, mrefs.forward)
		}
	}
	delete(ms.idx, zid)
	return toCheck
}

func (ms *memStore) deleteDeadSources(zid id.Zid, zi *zettelIndex) {
	// Must only be called if ms.mx is write-locked!
	for _, ref := range zi.dead {
		if drefs, ok := ms.dead[ref]; ok {
			drefs = remRef(drefs, zid)
			if len(drefs) > 0 {
				ms.dead[ref] = drefs
			} else {
				delete(ms.dead, ref)
			}
		}
	}
}

func (ms *memStore) deleteForwardBackward(zid id.Zid, zi *zettelIndex) id.Set {
	// Must only be called if ms.mx is write-locked!
	var toCheck id.Set
	for _, ref := range zi.forward {
		if fzi, ok := ms.idx[ref]; ok {
			fzi.backward = remRef(fzi.backward, zid)
		}
	}
	for _, ref := range zi.backward {
		if bzi, ok := ms.idx[ref]; ok {
			bzi.forward = remRef(bzi.forward, zid)
			if toCheck == nil {
				toCheck = id.NewSet()
			}
			toCheck[ref] = true
		}
	}
	return toCheck
}

func (ms *memStore) removeInverseMeta(zid id.Zid, key string, forward id.Slice) {
	// Must only be called if ms.mx is write-locked!
	for _, ref := range forward {
		bzi, ok := ms.idx[ref]
		if !ok || bzi.meta == nil {
			continue
		}
		bmr, ok := bzi.meta[key]
		if !ok {
			continue
		}
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

func (ms *memStore) ReadStats(st *index.StoreStats) {
	ms.mx.RLock()
	st.Zettel = len(ms.idx)
	st.Updates = ms.updates
	st.Words = uint64(len(ms.words))
	ms.mx.RUnlock()
}

func (ms *memStore) Write(w io.Writer) {
	ms.mx.RLock()
	defer ms.mx.RUnlock()

	zids := make(id.Slice, 0, len(ms.idx))
	for id := range ms.idx {
		zids = append(zids, id)
	}
	zids.Sort()
	for _, id := range zids {
		fmt.Fprintln(w, id)
		zi := ms.idx[id]
		fmt.Fprintln(w, "-", zi.dead)
		writeZidsLn(w, ">", zi.forward)
		writeZidsLn(w, "<", zi.backward)
		if zi.meta == nil {
			fmt.Fprintln(w, "*NIL")
		} else if len(zi.meta) == 0 {
			fmt.Fprintln(w, "*(0)")
		} else {
			for k, fb := range zi.meta {
				fmt.Fprintln(w, "*", k)
				writeZidsLn(w, "]", fb.forward)
				writeZidsLn(w, "[", fb.backward)
			}
		}
	}

	zids = make(id.Slice, 0, len(ms.dead))
	for id := range ms.dead {
		zids = append(zids, id)
	}
	zids.Sort()
	for _, id := range zids {
		fmt.Fprintln(w, "~", id, ms.dead[id])
	}
}

func writeZidsLn(w io.Writer, prefix string, zids id.Slice) {
	io.WriteString(w, prefix)
	for _, zid := range zids {
		io.WriteString(w, " ")
		w.Write(zid.Bytes())
	}
	fmt.Fprintln(w)
}
