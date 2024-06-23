//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2021-present Detlef Stern
//-----------------------------------------------------------------------------

// Package mapstore stored the index in main memory via a Go map.
package mapstore

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"

	"t73f.de/r/zsc/api"
	"t73f.de/r/zsc/maps"
	"zettelstore.de/z/box"
	"zettelstore.de/z/box/manager/store"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

type zettelData struct {
	meta      *meta.Meta // a local copy of the metadata, without computed keys
	dead      id.Slice   // list of dead references in this zettel
	forward   id.Slice   // list of forward references in this zettel
	backward  id.Slice   // list of zettel that reference with zettel
	otherRefs map[string]bidiRefs
	words     []string // list of words of this zettel
	urls      []string // list of urls of this zettel
}

type bidiRefs struct {
	forward  id.Slice
	backward id.Slice
}

type stringRefs map[string]id.Slice

type memStore struct {
	mx     sync.RWMutex
	intern map[string]string // map to intern strings
	idx    map[id.Zid]*zettelData
	dead   map[id.Zid]id.Slice // map dead refs where they occur
	words  stringRefs
	urls   stringRefs

	// Stats
	mxStats sync.Mutex
	updates uint64
}

// New returns a new memory-based index store.
func New() store.Store {
	return &memStore{
		intern: make(map[string]string, 1024),
		idx:    make(map[id.Zid]*zettelData),
		dead:   make(map[id.Zid]id.Slice),
		words:  make(stringRefs),
		urls:   make(stringRefs),
	}
}

func (ms *memStore) GetMeta(_ context.Context, zid id.Zid) (*meta.Meta, error) {
	ms.mx.RLock()
	defer ms.mx.RUnlock()
	if zi, found := ms.idx[zid]; found && zi.meta != nil {
		// zi.meta is nil, if zettel was referenced, but is not indexed yet.
		return zi.meta.Clone(), nil
	}
	return nil, box.ErrZettelNotFound{Zid: zid}
}

func (ms *memStore) Enrich(_ context.Context, m *meta.Meta) {
	if ms.doEnrich(m) {
		ms.mxStats.Lock()
		ms.updates++
		ms.mxStats.Unlock()
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
	back := removeOtherMetaRefs(m, zi.backward.Clone())
	if len(zi.backward) > 0 {
		m.Set(api.KeyBackward, zi.backward.String())
		updated = true
	}
	if len(zi.forward) > 0 {
		m.Set(api.KeyForward, zi.forward.String())
		back = remRefs(back, zi.forward)
		updated = true
	}
	for k, refs := range zi.otherRefs {
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
	return updated
}

// SearchEqual returns all zettel that contains the given exact word.
// The word must be normalized through Unicode NKFD, trimmed and not empty.
func (ms *memStore) SearchEqual(word string) *id.Set {
	ms.mx.RLock()
	defer ms.mx.RUnlock()
	result := id.NewSet()
	if refs, ok := ms.words[word]; ok {
		result.CopySlice(refs)
	}
	if refs, ok := ms.urls[word]; ok {
		result.CopySlice(refs)
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
func (ms *memStore) SearchPrefix(prefix string) *id.Set {
	ms.mx.RLock()
	defer ms.mx.RUnlock()
	result := ms.selectWithPred(prefix, strings.HasPrefix)
	l := len(prefix)
	if l > 14 {
		return result
	}
	maxZid, err := id.Parse(prefix + "99999999999999"[:14-l])
	if err != nil {
		return result
	}
	var minZid id.Zid
	if l < 14 && prefix == "0000000000000"[:l] {
		minZid = id.Zid(1)
	} else {
		minZid, err = id.Parse(prefix + "00000000000000"[:14-l])
		if err != nil {
			return result
		}
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
func (ms *memStore) SearchSuffix(suffix string) *id.Set {
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
	for range l {
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
func (ms *memStore) SearchContains(s string) *id.Set {
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

func (ms *memStore) selectWithPred(s string, pred func(string, string) bool) *id.Set {
	// Must only be called if ms.mx is read-locked!
	result := id.NewSet()
	for word, refs := range ms.words {
		if !pred(word, s) {
			continue
		}
		result.CopySlice(refs)
	}
	for u, refs := range ms.urls {
		if !pred(u, s) {
			continue
		}
		result.CopySlice(refs)
	}
	return result
}

func addBackwardZids(result *id.Set, zid id.Zid, zi *zettelData) {
	// Must only be called if ms.mx is read-locked!
	result.Add(zid)
	result.CopySlice(zi.backward)
	for _, mref := range zi.otherRefs {
		result.CopySlice(mref.backward)
	}
}

func removeOtherMetaRefs(m *meta.Meta, back id.Slice) id.Slice {
	for _, p := range m.PairsRest() {
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

func (ms *memStore) UpdateReferences(_ context.Context, zidx *store.ZettelIndex) *id.Set {
	ms.mx.Lock()
	defer ms.mx.Unlock()
	m := ms.makeMeta(zidx)
	zi, ziExist := ms.idx[zidx.Zid]
	if !ziExist || zi == nil {
		zi = &zettelData{}
		ziExist = false
	}

	// Is this zettel an old dead reference mentioned in other zettel?
	var toCheck *id.Set
	if refs, ok := ms.dead[zidx.Zid]; ok {
		// These must be checked later again
		toCheck = id.NewSet(refs...)
		delete(ms.dead, zidx.Zid)
	}

	zi.meta = m
	ms.updateDeadReferences(zidx, zi)
	ids := ms.updateForwardBackwardReferences(zidx, zi)
	toCheck = toCheck.Copy(ids)
	ids = ms.updateMetadataReferences(zidx, zi)
	toCheck = toCheck.Copy(ids)
	zi.words = updateStrings(zidx.Zid, ms.words, zi.words, zidx.GetWords())
	zi.urls = updateStrings(zidx.Zid, ms.urls, zi.urls, zidx.GetUrls())

	// Check if zi must be inserted into ms.idx
	if !ziExist {
		ms.idx[zidx.Zid] = zi
	}

	return toCheck
}

var internableKeys = map[string]bool{
	api.KeyRole:      true,
	api.KeySyntax:    true,
	api.KeyFolgeRole: true,
	api.KeyLang:      true,
	api.KeyReadOnly:  true,
}

func isInternableValue(key string) bool {
	if internableKeys[key] {
		return true
	}
	return strings.HasSuffix(key, meta.SuffixKeyRole)
}

func (ms *memStore) internString(s string) string {
	if is, found := ms.intern[s]; found {
		return is
	}
	ms.intern[s] = s
	return s
}

func (ms *memStore) makeMeta(zidx *store.ZettelIndex) *meta.Meta {
	origM := zidx.GetMeta()
	copyM := meta.New(origM.Zid)
	for _, p := range origM.Pairs() {
		key := ms.internString(p.Key)
		if isInternableValue(key) {
			copyM.Set(key, ms.internString(p.Value))
		} else if key == api.KeyBoxNumber || !meta.IsComputed(key) {
			copyM.Set(key, p.Value)
		}
	}
	return copyM
}

func (ms *memStore) updateDeadReferences(zidx *store.ZettelIndex, zi *zettelData) {
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

func (ms *memStore) updateForwardBackwardReferences(zidx *store.ZettelIndex, zi *zettelData) *id.Set {
	// Must only be called if ms.mx is write-locked!
	brefs := zidx.GetBackRefs()
	newRefs, remRefs := refsDiff(brefs, zi.forward)
	zi.forward = brefs

	var toCheck *id.Set
	for _, ref := range remRefs {
		bzi := ms.getOrCreateEntry(ref)
		bzi.backward = remRef(bzi.backward, zidx.Zid)
		if bzi.meta == nil {
			toCheck = toCheck.Add(ref)
		}
	}
	for _, ref := range newRefs {
		bzi := ms.getOrCreateEntry(ref)
		bzi.backward = addRef(bzi.backward, zidx.Zid)
		if bzi.meta == nil {
			toCheck = toCheck.Add(ref)
		}
	}
	return toCheck
}

func (ms *memStore) updateMetadataReferences(zidx *store.ZettelIndex, zi *zettelData) *id.Set {
	// Must only be called if ms.mx is write-locked!
	inverseRefs := zidx.GetInverseRefs()
	for key, mr := range zi.otherRefs {
		if _, ok := inverseRefs[key]; ok {
			continue
		}
		ms.removeInverseMeta(zidx.Zid, key, mr.forward)
	}
	if zi.otherRefs == nil {
		zi.otherRefs = make(map[string]bidiRefs)
	}
	var toCheck *id.Set
	for key, mrefs := range inverseRefs {
		mr := zi.otherRefs[key]
		newRefs, remRefs := refsDiff(mrefs, mr.forward)
		mr.forward = mrefs
		zi.otherRefs[key] = mr

		for _, ref := range newRefs {
			bzi := ms.getOrCreateEntry(ref)
			if bzi.otherRefs == nil {
				bzi.otherRefs = make(map[string]bidiRefs)
			}
			bmr := bzi.otherRefs[key]
			bmr.backward = addRef(bmr.backward, zidx.Zid)
			bzi.otherRefs[key] = bmr
			if bzi.meta == nil {
				toCheck = toCheck.Add(ref)
			}
		}
		ms.removeInverseMeta(zidx.Zid, key, remRefs)
	}
	return toCheck
}

func updateStrings(zid id.Zid, srefs stringRefs, prev []string, next store.WordSet) []string {
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

func (ms *memStore) getOrCreateEntry(zid id.Zid) *zettelData {
	// Must only be called if ms.mx is write-locked!
	if zi, ok := ms.idx[zid]; ok {
		return zi
	}
	zi := &zettelData{}
	ms.idx[zid] = zi
	return zi
}

func (ms *memStore) RenameZettel(_ context.Context, curZid, newZid id.Zid) *id.Set {
	ms.mx.Lock()
	defer ms.mx.Unlock()

	curZi, curFound := ms.idx[curZid]
	_, newFound := ms.idx[newZid]
	if !curFound || newFound {
		return nil
	}
	newZi := &zettelData{
		meta:      copyMeta(curZi.meta, newZid),
		dead:      ms.copyDeadReferences(curZi.dead),
		forward:   ms.copyForward(curZi.forward, newZid),
		backward:  nil, // will be done through tocheck
		otherRefs: nil, // TODO: check if this will be done through toCheck
		words:     copyStrings(ms.words, curZi.words, newZid),
		urls:      copyStrings(ms.urls, curZi.urls, newZid),
	}

	ms.idx[newZid] = newZi
	toCheck := ms.doDeleteZettel(curZid)
	toCheck = toCheck.CopySlice(ms.dead[newZid])
	delete(ms.dead, newZid)
	toCheck = toCheck.Add(newZid) // should update otherRefs
	return toCheck
}
func copyMeta(m *meta.Meta, newZid id.Zid) *meta.Meta {
	result := m.Clone()
	result.Zid = newZid
	return result
}
func (ms *memStore) copyDeadReferences(curDead id.Slice) id.Slice {
	// Must only be called if ms.mx is write-locked!
	if l := len(curDead); l > 0 {
		result := make(id.Slice, l)
		for i, ref := range curDead {
			result[i] = ref
			ms.dead[ref] = addRef(ms.dead[ref], ref)
		}
		return result
	}
	return nil
}
func (ms *memStore) copyForward(curForward id.Slice, newZid id.Zid) id.Slice {
	// Must only be called if ms.mx is write-locked!
	if l := len(curForward); l > 0 {
		result := make(id.Slice, l)
		for i, ref := range curForward {
			result[i] = ref
			if fzi, found := ms.idx[ref]; found {
				fzi.backward = addRef(fzi.backward, newZid)
			}
		}
		return result
	}
	return nil
}
func copyStrings(msStringMap stringRefs, curStrings []string, newZid id.Zid) []string {
	// Must only be called if ms.mx is write-locked!
	if l := len(curStrings); l > 0 {
		result := make([]string, l)
		for i, s := range curStrings {
			result[i] = s
			msStringMap[s] = addRef(msStringMap[s], newZid)
		}
		return result
	}
	return nil
}

func (ms *memStore) DeleteZettel(_ context.Context, zid id.Zid) *id.Set {
	ms.mx.Lock()
	defer ms.mx.Unlock()
	return ms.doDeleteZettel(zid)
}

func (ms *memStore) doDeleteZettel(zid id.Zid) *id.Set {
	// Must only be called if ms.mx is write-locked!
	zi, ok := ms.idx[zid]
	if !ok {
		return nil
	}

	ms.deleteDeadSources(zid, zi)
	toCheck := ms.deleteForwardBackward(zid, zi)
	for key, mrefs := range zi.otherRefs {
		ms.removeInverseMeta(zid, key, mrefs.forward)
	}
	deleteStrings(ms.words, zi.words, zid)
	deleteStrings(ms.urls, zi.urls, zid)
	delete(ms.idx, zid)
	return toCheck
}

func (ms *memStore) deleteDeadSources(zid id.Zid, zi *zettelData) {
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

func (ms *memStore) deleteForwardBackward(zid id.Zid, zi *zettelData) *id.Set {
	// Must only be called if ms.mx is write-locked!
	for _, ref := range zi.forward {
		if fzi, ok := ms.idx[ref]; ok {
			fzi.backward = remRef(fzi.backward, zid)
		}
	}
	var toCheck *id.Set
	for _, ref := range zi.backward {
		if bzi, ok := ms.idx[ref]; ok {
			bzi.forward = remRef(bzi.forward, zid)
			toCheck = toCheck.Add(ref)
		}
	}
	return toCheck
}

func (ms *memStore) removeInverseMeta(zid id.Zid, key string, forward id.Slice) {
	// Must only be called if ms.mx is write-locked!
	for _, ref := range forward {
		bzi, ok := ms.idx[ref]
		if !ok || bzi.otherRefs == nil {
			continue
		}
		bmr, ok := bzi.otherRefs[key]
		if !ok {
			continue
		}
		bmr.backward = remRef(bmr.backward, zid)
		if len(bmr.backward) > 0 || len(bmr.forward) > 0 {
			bzi.otherRefs[key] = bmr
		} else {
			delete(bzi.otherRefs, key)
			if len(bzi.otherRefs) == 0 {
				bzi.otherRefs = nil
			}
		}
	}
}

func deleteStrings(msStringMap stringRefs, curStrings []string, zid id.Zid) {
	// Must only be called if ms.mx is write-locked!
	for _, word := range curStrings {
		refs, ok := msStringMap[word]
		if !ok {
			continue
		}
		refs2 := remRef(refs, zid)
		if len(refs2) == 0 {
			delete(msStringMap, word)
			continue
		}
		msStringMap[word] = refs2
	}
}

func (ms *memStore) ReadStats(st *store.Stats) {
	ms.mx.RLock()
	st.Zettel = len(ms.idx)
	st.Words = uint64(len(ms.words))
	st.Urls = uint64(len(ms.urls))
	ms.mx.RUnlock()
	ms.mxStats.Lock()
	st.Updates = ms.updates
	ms.mxStats.Unlock()
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

		otherRefs := make([]string, 0, len(zi.otherRefs))
		for k := range zi.otherRefs {
			otherRefs = append(otherRefs, k)
		}
		slices.Sort(otherRefs)
		for _, k := range otherRefs {
			fmt.Fprintln(w, "* Meta", k)
			dumpZids(w, "** Forward:", zi.otherRefs[k].forward)
			dumpZids(w, "** Backward:", zi.otherRefs[k].backward)
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
		slices.Sort(sl)
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
	for _, s := range maps.Keys(srefs) {
		fmt.Fprintf(w, "; %s%s%s\n", preString, s, postString)
		fmt.Fprintln(w, ":", srefs[s])
	}
}
