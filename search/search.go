//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package search provides a zettel search.
package search

import (
	"math/rand"
	"sort"
	"sync"

	"zettelstore.de/z/domain/meta"
)

// MetaMatchFunc is a function determine whethe some metadata should be filtered or not.
type MetaMatchFunc func(*meta.Meta) bool

// Filter specifies a mechanism for selecting zettel.
type Filter struct {
	mx        sync.RWMutex          // Protects other attributes
	preFilter MetaMatchFunc         // Filter to be executed first
	tags      map[string][]expValue // Expected values for a tag
	search    []expValue            // Search string
	negate    bool                  // Negate the result of the whole filtering process
	compiled  MetaMatchFunc         // Compiled function that implements above spec data
}

type expValue struct {
	value  string
	negate bool
}

// AddExpr adds a match expression to the filter.
func (f *Filter) AddExpr(key, val string, negate bool) *Filter {
	if f == nil {
		f = new(Filter)
	}
	f.mx.Lock()
	defer f.mx.Unlock()
	if key == "" {
		f.search = append(f.search, expValue{value: val, negate: negate})
	} else if f.tags == nil {
		f.tags = map[string][]expValue{key: {{value: val, negate: negate}}}
	} else {
		f.tags[key] = append(f.tags[key], expValue{value: val, negate: negate})
	}
	return f
}

// SetNegate changes the filter to reverse its selection.
func (f *Filter) SetNegate() *Filter {
	if f == nil {
		f = new(Filter)
	}
	f.mx.Lock()
	defer f.mx.Unlock()
	f.negate = true
	return f
}

// AddPreFilter adds the pre-filter selection predicate.
func (f *Filter) AddPreFilter(preFilter MetaMatchFunc) *Filter {
	if f == nil {
		f = new(Filter)
	}
	f.mx.Lock()
	defer f.mx.Unlock()
	if pre := f.preFilter; pre == nil {
		f.preFilter = preFilter
	} else {
		f.preFilter = func(m *meta.Meta) bool {
			return preFilter(m) && pre(m)
		}
	}
	return f
}

// HasComputedMetaKey returns true, if the filter references a metadata key which
// a computed value.
func (f *Filter) HasComputedMetaKey() bool {
	if f == nil {
		return false
	}
	f.mx.RLock()
	defer f.mx.RUnlock()
	for key := range f.tags {
		if meta.IsComputed(key) {
			return true
		}
	}
	return false
}

// Match checks whether the given meta matches the filter specification.
func (f *Filter) Match(m *meta.Meta) bool {
	if f == nil {
		return true
	}
	f.mx.Lock()
	defer f.mx.Unlock()
	if pre := f.preFilter; pre != nil {
		if !pre(m) {
			return false
		}
	}
	if f.compiled == nil {
		f.compiled = compileFilter(f)
	}
	return f.compiled(m) != f.negate
}

// Sorter specifies ordering and limiting a sequnce of meta data.
type Sorter struct {
	Order      string // Name of meta key. None given: use "id"
	Descending bool   // Sort by order, but descending
	Offset     int    // <= 0: no offset
	Limit      int    // <= 0: no limit
}

// Ensure makes sure that there is a sorter object.
func (s *Sorter) Ensure() *Sorter {
	if s == nil {
		s = new(Sorter)
	}
	return s
}

// RandomOrder is a pseudo metadata key that selects a random order.
const RandomOrder = "_random"

// Sort applies the sorter to the slice of meta data.
func (s *Sorter) Sort(metaList []*meta.Meta) []*meta.Meta {
	if len(metaList) == 0 {
		return metaList
	}

	if s == nil {
		sort.Slice(metaList, func(i, j int) bool { return metaList[i].Zid > metaList[j].Zid })
		return metaList
	}

	if s.Order == "" {
		sort.Slice(metaList, createSortFunc(meta.KeyID, true, metaList))
	} else if s.Order == RandomOrder {
		rand.Shuffle(len(metaList), func(i, j int) {
			metaList[i], metaList[j] = metaList[j], metaList[i]
		})
	} else {
		sort.Slice(metaList, createSortFunc(s.Order, s.Descending, metaList))
	}

	if s.Offset > 0 {
		if s.Offset > len(metaList) {
			return nil
		}
		metaList = metaList[s.Offset:]
	}
	if s.Limit > 0 && s.Limit < len(metaList) {
		metaList = metaList[:s.Limit]
	}
	return metaList
}
