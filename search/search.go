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
	"strings"
	"sync"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

// Searcher is used to select zettel identifier based on search criteria.
type Searcher interface {
	// Select all zettel that contains the given exact word.
	// The word must be normalized through Unicode NKFD, trimmed and not empty.
	SearchEqual(word string) id.Set

	// Select all zettel that have a word with the given prefix.
	// The prefix must be normalized through Unicode NKFD, trimmed and not empty.
	SearchPrefix(prefix string) id.Set

	// Select all zettel that have a word with the given suffix.
	// The suffix must be normalized through Unicode NKFD, trimmed and not empty.
	SearchSuffix(suffix string) id.Set

	// Select all zettel that contains the given string.
	// The string must be normalized through Unicode NKFD, trimmed and not empty.
	SearchContains(s string) id.Set
}

// MetaMatchFunc is a function determine whethe some metadata should be selected or not.
type MetaMatchFunc func(*meta.Meta) bool

// Search specifies a mechanism for selecting zettel.
type Search struct {
	mx sync.RWMutex // Protects other attributes

	// Fields to be used for selecting
	preMatch MetaMatchFunc // Match that must be true
	tags     expTagValues  // Expected values for a tag
	search   []expValue    // Search string
	negate   bool          // Negate the result of the whole selecting process

	// Fields to be used for sorting
	order      string // Name of meta key. None given: use "id"
	descending bool   // Sort by order, but descending
	offset     int    // <= 0: no offset
	limit      int    // <= 0: no limit
}

type expTagValues map[string][]expValue

// RandomOrder is a pseudo metadata key that selects a random order.
const RandomOrder = "_random"

type compareOp uint8

const (
	cmpUnknown compareOp = iota
	cmpDefault
	cmpNotDefault
	cmpEqual
	cmpNotEqual
	cmpPrefix
	cmpNoPrefix
	cmpSuffix
	cmpNoSuffix
	cmpContains
	cmpNotContains
)

var negateMap = map[compareOp]compareOp{
	cmpUnknown:     cmpUnknown,
	cmpDefault:     cmpNotDefault,
	cmpNotDefault:  cmpDefault,
	cmpEqual:       cmpNotEqual,
	cmpNotEqual:    cmpEqual,
	cmpPrefix:      cmpNoPrefix,
	cmpNoPrefix:    cmpPrefix,
	cmpSuffix:      cmpNoSuffix,
	cmpNoSuffix:    cmpSuffix,
	cmpContains:    cmpNotContains,
	cmpNotContains: cmpContains,
}

func (op compareOp) negate() compareOp {
	return negateMap[op]
}

type expValue struct {
	value  string
	op     compareOp
	negate bool
}

// AddExpr adds a match expression to the search.
func (s *Search) AddExpr(key, val string) *Search {
	val, negate, op := parseOp(strings.TrimSpace(val))
	if s == nil {
		s = new(Search)
	}
	s.mx.Lock()
	defer s.mx.Unlock()
	if key == "" {
		s.search = append(s.search, expValue{value: val, op: op, negate: negate})
	} else {
		if s.tags == nil {
			s.tags = expTagValues{key: {{value: val, op: op, negate: negate}}}
		} else {
			s.tags[key] = append(s.tags[key], expValue{value: val, op: op, negate: negate})
		}
	}
	return s
}

func parseOp(s string) (r string, negate bool, op compareOp) {
	if s == "" {
		return s, false, cmpDefault
	}
	if s[0] == '\\' {
		return s[1:], false, cmpDefault
	}
	if s[0] == '!' {
		negate = true
		s = s[1:]
	}
	if s == "" {
		return s, negate, cmpDefault
	}
	if s[0] == '\\' {
		return s[1:], negate, cmpDefault
	}
	switch s[0] {
	case ':':
		return s[1:], negate, cmpDefault
	case '=':
		return s[1:], negate, cmpEqual
	case '>':
		return s[1:], negate, cmpPrefix
	case '<':
		return s[1:], negate, cmpSuffix
	case '~':
		return s[1:], negate, cmpContains
	}
	return s, negate, cmpDefault
}

// SetNegate changes the search to reverse its selection.
func (s *Search) SetNegate() *Search {
	if s == nil {
		s = new(Search)
	}
	s.mx.Lock()
	defer s.mx.Unlock()
	s.negate = true
	return s
}

// AddPreMatch adds the pre-selection predicate.
func (s *Search) AddPreMatch(preMatch MetaMatchFunc) *Search {
	if s == nil {
		s = new(Search)
	}
	s.mx.Lock()
	defer s.mx.Unlock()
	if pre := s.preMatch; pre == nil {
		s.preMatch = preMatch
	} else {
		s.preMatch = func(m *meta.Meta) bool {
			return preMatch(m) && pre(m)
		}
	}
	return s
}

// AddOrder adds the given order to the search object.
func (s *Search) AddOrder(key string, descending bool) *Search {
	if s == nil {
		s = new(Search)
	}
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.order != "" {
		panic("order field already set: " + s.order)
	}
	s.order = key
	s.descending = descending
	return s
}

// SetOffset sets the given offset of the search object.
func (s *Search) SetOffset(offset int) *Search {
	if s == nil {
		s = new(Search)
	}
	s.mx.Lock()
	defer s.mx.Unlock()
	if offset < 0 {
		offset = 0
	}
	s.offset = offset
	return s
}

// GetOffset returns the current offset value.
func (s *Search) GetOffset() int {
	if s == nil {
		return 0
	}
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.offset
}

// SetLimit sets the given limit of the search object.
func (s *Search) SetLimit(limit int) *Search {
	if s == nil {
		s = new(Search)
	}
	s.mx.Lock()
	defer s.mx.Unlock()
	if limit < 0 {
		limit = 0
	}
	s.limit = limit
	return s
}

// GetLimit returns the current offset value.
func (s *Search) GetLimit() int {
	if s == nil {
		return 0
	}
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.limit
}

// EnrichNeeded returns true, if the search references a metadata key that
// is calculated via metadata enrichments. In most cases this is a computed
// value. Metadata "tags" is an exception to this rule.
func (s *Search) EnrichNeeded() bool {
	if s == nil {
		return false
	}
	s.mx.RLock()
	defer s.mx.RUnlock()
	for key := range s.tags {
		if meta.IsComputed(key) || key == api.KeyTags {
			return true
		}
	}
	if order := s.order; order != "" && (meta.IsComputed(order) || order == api.KeyTags) {
		return true
	}
	return false
}

// CompileMatch returns a function to match meta data based on select specification.
func (s *Search) CompileMatch(searcher Searcher) MetaMatchFunc {
	if s == nil {
		return selectNone
	}
	s.mx.Lock()
	defer s.mx.Unlock()

	compMeta := compileSelect(s.tags)
	compSearch := compileFullSearch(searcher, s.search)
	if preMatch := s.preMatch; preMatch != nil {
		return compilePreMatch(preMatch, compMeta, compSearch, s.negate)
	}
	return compileNoPreMatch(compMeta, compSearch, s.negate)
}

func selectNone(*meta.Meta) bool { return true }

func compilePreMatch(preMatch, compMeta, compSearch MetaMatchFunc, negate bool) MetaMatchFunc {
	if compMeta == nil {
		if compSearch == nil {
			return preMatch
		}
		if negate {
			return func(m *meta.Meta) bool { return preMatch(m) && !compSearch(m) }
		}
		return func(m *meta.Meta) bool { return preMatch(m) && compSearch(m) }
	}
	if compSearch == nil {
		if negate {
			return func(m *meta.Meta) bool { return preMatch(m) && !compMeta(m) }
		}
		return func(m *meta.Meta) bool { return preMatch(m) && compMeta(m) }
	}
	if negate {
		return func(m *meta.Meta) bool { return preMatch(m) && (!compMeta(m) || !compSearch(m)) }
	}
	return func(m *meta.Meta) bool { return preMatch(m) && compMeta(m) && compSearch(m) }
}

func compileNoPreMatch(compMeta, compSearch MetaMatchFunc, negate bool) MetaMatchFunc {
	if compMeta == nil {
		if compSearch == nil {
			if negate {
				return func(m *meta.Meta) bool { return false }
			}
			return selectNone
		}
		if negate {
			return func(m *meta.Meta) bool { return !compSearch(m) }
		}
		return compSearch
	}
	if compSearch == nil {
		if negate {
			return func(m *meta.Meta) bool { return !compMeta(m) }
		}
		return compMeta
	}
	if negate {
		return func(m *meta.Meta) bool { return !compMeta(m) || !compSearch(m) }
	}
	return func(m *meta.Meta) bool { return compMeta(m) && compSearch(m) }
}

// Sort applies the sorter to the slice of meta data.
func (s *Search) Sort(metaList []*meta.Meta) []*meta.Meta {
	if len(metaList) == 0 {
		return metaList
	}

	if s == nil {
		sort.Slice(metaList, func(i, j int) bool { return metaList[i].Zid > metaList[j].Zid })
		return metaList
	}

	if s.order == "" {
		sort.Slice(metaList, createSortFunc(api.KeyID, true, metaList))
	} else if s.order == RandomOrder {
		rand.Shuffle(len(metaList), func(i, j int) {
			metaList[i], metaList[j] = metaList[j], metaList[i]
		})
	} else {
		sort.Slice(metaList, createSortFunc(s.order, s.descending, metaList))
	}

	if s.offset > 0 {
		if s.offset > len(metaList) {
			return nil
		}
		metaList = metaList[s.offset:]
	}
	if s.limit > 0 && s.limit < len(metaList) {
		metaList = metaList[:s.limit]
	}
	return metaList
}
