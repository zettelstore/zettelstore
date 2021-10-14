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
	"sort"
	"strings"
	"sync"

	"zettelstore.de/c/api"
	"zettelstore.de/z/box/manager/store"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
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
	words    []string
	urls     []string
	itags    string // Inline tags
}

func (zi *zettelIndex) isEmpty() bool {
	if len(zi.forward) > 0 || len(zi.backward) > 0 || len(zi.dead) > 0 || len(zi.words) > 0 {
		return false
	}
	return len(zi.meta) == 0
}

type stringRefs map[string]id.Slice

type memStore struct {
	mx    sync.RWMutex
	idx   map[id.Zid]*zettelIndex
	dead  map[id.Zid]id.Slice // map dead refs where they occur
	words stringRefs
	urls  stringRefs

	// Stats
	updates uint64
}

// New returns a new memory-based index store.
func New() store.Store {
	return &memStore{
		idx:   make(map[id.Zid]*zettelIndex),
		dead:  make(map[id.Zid]id.Slice),
		words: make(stringRefs),
		urls:  make(stringRefs),
	}
}

func (ms *memStore) Enrich(_ context.Context, m *meta.Meta) {
	if ms.doEnrich(m) {
		ms.mx.Lock()
		ms.updates++
		ms.mx.Unlock()
	}
}

func (ms *memStore) doEnrich(m *meta.Meta) bool {
	ms.mx.RLock()
	defer ms.mx.RUnlock()
	zi, ok := ms.idx[m.Zid]
	if !ok {
		return false
	}
	var updated bool
	if len(zi.dead) > 0 {
		m.Set(api.KeyDead, zi.dead.String())
		updated = true
	}
	back := removeOtherMetaRefs(m, zi.backward.Copy())
	if len(zi.backward) > 0 {
		m.Set(api.KeyBackward, zi.backward.String())
		updated = true
	}
	if len(zi.forward) > 0 {
		m.Set(api.KeyForward, zi.forward.String())
		back = remRefs(back, zi.forward)
		updated = true
	}
	for k, refs := range zi.meta {
		if len(refs.backward) > 0 {
			m.Set(k, refs.backward.String())
			back = remRefs(back, refs.backward)
			updated = true
		}
	}
	if len(back) > 0 {
		m.Set(api.KeyBack, back.String())
		updated = true
	}
	if itags := zi.itags; itags != "" {
		m.Set(api.KeyContentTags, itags)
		if tags, ok := m.Get(api.KeyTags); ok {
			m.Set(api.KeyAllTags, tags+" "+itags)
		} else {
			m.Set(api.KeyAllTags, itags)
		}
		updated = true
	} else if tags, ok := m.Get(api.KeyTags); ok {
		m.Set(api.KeyAllTags, tags)
		updated = true
	}
	return updated
}

// SearchEqual returns all zettel that contains the given exact word.
// The word must be normalized through Unicode NKFD, trimmed and not empty.
func (ms *memStore) SearchEqual(word string) id.Set {
	ms.mx.RLock()
	defer ms.mx.RUnlock()
	result := id.NewSet()
	if refs, ok := ms.words[word]; ok {
		result.AddSlice(refs)
	}
	if refs, ok := ms.urls[word]; ok {
		result.AddSlice(refs)
	}
	zid, err := id.Parse(word)
	if err != nil {
		return result
	}
	zi, ok := ms.idx[zid]
	if !ok {
		return result
	}

	addBackwardZids(result, zid, zi)
	return result
}

// SearchPrefix returns all zettel that have a word with the given prefix.
// The prefix must be normalized through Unicode NKFD, trimmed and not empty.
func (ms *memStore) SearchPrefix(prefix string) id.Set {
	ms.mx.RLock()
	defer ms.mx.RUnlock()
	result := ms.selectWithPred(prefix, strings.HasPrefix)
	l := len(prefix)
	if l > 14 {
		return result
	}
	minZid, err := id.Parse(prefix + "00000000000000"[:14-l])
	if err != nil {
		return result
	}
	maxZid, err := id.Parse(prefix + "99999999999999"[:14-l])
	if err != nil {
		return result
	}
	for zid, zi := range ms.idx {
		if minZid <= zid && zid <= maxZid {
			addBackwardZids(result, zid, zi)
		}
	}
	return result
}

// SearchSuffix returns all zettel that have a word with the given suffix.
// The suffix must be normalized through Unicode NKFD, trimmed and not empty.
func (ms *memStore) SearchSuffix(suffix string) id.Set {
	ms.mx.RLock()
	defer ms.mx.RUnlock()
	result := ms.selectWithPred(suffix, strings.HasSuffix)
	l := len(suffix)
	if l > 14 {
		return result
	}
	val, err := id.ParseUint(suffix)
	if err != nil {
		return result
	}
	modulo := uint64(1)
	for i := 0; i < l; i++ {
		modulo *= 10
	}
	for zid, zi := range ms.idx {
		if uint64(zid)%modulo == val {
			addBackwardZids(result, zid, zi)
		}
	}
	return result
}

// SearchContains returns all zettel that contains the given string.
// The string must be normalized through Unicode NKFD, trimmed and not empty.
func (ms *memStore) SearchContains(s string) id.Set {
	ms.mx.RLock()
	defer ms.mx.RUnlock()
	result := ms.selectWithPred(s, strings.Contains)
	if len(s) > 14 {
		return result
	}
	if _, err := id.ParseUint(s); err != nil {
		return result
	}
	for zid, zi := range ms.idx {
		if strings.Contains(zid.String(), s) {
			addBackwardZids(result, zid, zi)
		}
	}
	return result
}

func (ms *memStore) selectWithPred(s string, pred func(string, string) bool) id.Set {
	// Must only be called if ms.mx is read-locked!
	result := id.NewSet()
	for word, refs := range ms.words {
		if !pred(word, s) {
			continue
		}
		result.AddSlice(refs)
	}
	for u, refs := range ms.urls {
		if !pred(u, s) {
			continue
		}
		result.AddSlice(refs)
	}
	return result
}

func addBackwardZids(result id.Set, zid id.Zid, zi *zettelIndex) {
	// Must only be called if ms.mx is read-locked!
	result[zid] = true
	result.AddSlice(zi.backward)
	for _, mref := range zi.meta {
		result.AddSlice(mref.backward)
	}
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

func (ms *memStore) UpdateReferences(_ context.Context, zidx *store.ZettelIndex) id.Set {
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
	zi.words = updateWordSet(zidx.Zid, ms.words, zi.words, zidx.GetWords())
	zi.urls = updateWordSet(zidx.Zid, ms.urls, zi.urls, zidx.GetUrls())
	zi.itags = setITags(zidx.GetITags())

	// Check if zi must be inserted into ms.idx
	if !ziExist && !zi.isEmpty() {
		ms.idx[zidx.Zid] = zi
	}

	return toCheck
}

func (ms *memStore) updateDeadReferences(zidx *store.ZettelIndex, zi *zettelIndex) {
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

func (ms *memStore) updateForwardBackwardReferences(zidx *store.ZettelIndex, zi *zettelIndex) {
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

func (ms *memStore) updateMetadataReferences(zidx *store.ZettelIndex, zi *zettelIndex) {
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

func updateWordSet(zid id.Zid, srefs stringRefs, prev []string, next store.WordSet) []string {
	// Must only be called if ms.mx is write-locked!
	newWords, removeWords := next.Diff(prev)
	for _, word := range newWords {
		if refs, ok := srefs[word]; ok {
			srefs[word] = addRef(refs, zid)
			continue
		}
		srefs[word] = id.Slice{zid}
	}
	for _, word := range removeWords {
		refs, ok := srefs[word]
		if !ok {
			continue
		}
		refs2 := remRef(refs, zid)
		if len(refs2) == 0 {
			delete(srefs, word)
			continue
		}
		srefs[word] = refs2
	}
	return next.Words()
}

func setITags(next store.WordSet) string {
	itags := next.Words()
	if len(itags) == 0 {
		return ""
	}
	sort.Strings(itags)
	return strings.Join(itags, " ")
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

func (ms *memStore) DeleteZettel(_ context.Context, zid id.Zid) id.Set {
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
	ms.deleteWords(zid, zi.words)
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

func (ms *memStore) deleteWords(zid id.Zid, words []string) {
	// Must only be called if ms.mx is write-locked!
	for _, word := range words {
		refs, ok := ms.words[word]
		if !ok {
			continue
		}
		refs2 := remRef(refs, zid)
		if len(refs2) == 0 {
			delete(ms.words, word)
			continue
		}
		ms.words[word] = refs2
	}
}

func (ms *memStore) ReadStats(st *store.Stats) {
	ms.mx.RLock()
	st.Zettel = len(ms.idx)
	st.Updates = ms.updates
	st.Words = uint64(len(ms.words))
	st.Urls = uint64(len(ms.urls))
	ms.mx.RUnlock()
}

func (ms *memStore) Dump(w io.Writer) {
	ms.mx.RLock()
	defer ms.mx.RUnlock()

	io.WriteString(w, "=== Dump\n")
	ms.dumpIndex(w)
	ms.dumpDead(w)
	dumpStringRefs(w, "Words", "", "", ms.words)
	dumpStringRefs(w, "URLs", "[[", "]]", ms.urls)
}

func (ms *memStore) dumpIndex(w io.Writer) {
	if len(ms.idx) == 0 {
		return
	}
	io.WriteString(w, "==== Zettel Index\n")
	zids := make(id.Slice, 0, len(ms.idx))
	for id := range ms.idx {
		zids = append(zids, id)
	}
	zids.Sort()
	for _, id := range zids {
		fmt.Fprintln(w, "=====", id)
		zi := ms.idx[id]
		if len(zi.dead) > 0 {
			fmt.Fprintln(w, "* Dead:", zi.dead)
		}
		dumpZids(w, "* Forward:", zi.forward)
		dumpZids(w, "* Backward:", zi.backward)
		for k, fb := range zi.meta {
			fmt.Fprintln(w, "* Meta", k)
			dumpZids(w, "** Forward:", fb.forward)
			dumpZids(w, "** Backward:", fb.backward)
		}
		dumpStrings(w, "* Words", "", "", zi.words)
		dumpStrings(w, "* URLs", "[[", "]]", zi.urls)
	}
}

func (ms *memStore) dumpDead(w io.Writer) {
	if len(ms.dead) == 0 {
		return
	}
	fmt.Fprintf(w, "==== Dead References\n")
	zids := make(id.Slice, 0, len(ms.dead))
	for id := range ms.dead {
		zids = append(zids, id)
	}
	zids.Sort()
	for _, id := range zids {
		fmt.Fprintln(w, ";", id)
		fmt.Fprintln(w, ":", ms.dead[id])
	}
}

func dumpZids(w io.Writer, prefix string, zids id.Slice) {
	if len(zids) > 0 {
		io.WriteString(w, prefix)
		for _, zid := range zids {
			io.WriteString(w, " ")
			w.Write(zid.Bytes())
		}
		fmt.Fprintln(w)
	}
}

func dumpStrings(w io.Writer, title, preString, postString string, slice []string) {
	if len(slice) > 0 {
		sl := make([]string, len(slice))
		copy(sl, slice)
		sort.Strings(sl)
		fmt.Fprintln(w, title)
		for _, s := range sl {
			fmt.Fprintf(w, "** %s%s%s\n", preString, s, postString)
		}
	}

}

func dumpStringRefs(w io.Writer, title, preString, postString string, srefs stringRefs) {
	if len(srefs) == 0 {
		return
	}
	fmt.Fprintln(w, "====", title)
	slice := make([]string, 0, len(srefs))
	for s := range srefs {
		slice = append(slice, s)
	}
	sort.Strings(slice)
	for _, s := range slice {
		fmt.Fprintf(w, "; %s%s%s\n", preString, s, postString)
		fmt.Fprintln(w, ":", srefs[s])
	}
}
