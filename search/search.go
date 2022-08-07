//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
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

// RetrieveFunc retrieves the index based on a Search.
type RetrieveFunc func() id.Set

// RetrievePredicate returns true, if the given Zid is contained in the (full-text) search.
type RetrievePredicate func(id.Zid) bool

// Search specifies a mechanism for selecting zettel.
type Search struct {
	mx sync.RWMutex // Protects other attributes

	// Fields to be used for selecting
	preMatch MetaMatchFunc // Match that must be true
	mvals    expMetaValues // Expected values for a meta datum
	search   []expValue    // Search string
	negate   bool          // Negate the result of the whole selecting process

	// Fields to be used for sorting
	order      string // Name of meta key. None given: use "id"
	descending bool   // Sort by order, but descending
	offset     int    // <= 0: no offset
	limit      int    // <= 0: no limit
}

type expMetaValues map[string][]expValue

func createIfNeeded(s *Search) *Search {
	if s == nil {
		return new(Search)
	}
	return s
}

// Clone the search value.
func (s *Search) Clone() *Search {
	if s == nil {
		return nil
	}
	c := new(Search)
	c.preMatch = s.preMatch
	c.mvals = make(expMetaValues, len(s.mvals))
	for k, v := range s.mvals {
		c.mvals[k] = v
	}
	c.search = append([]expValue{}, s.search...)
	c.negate = s.negate
	c.order = s.order
	c.descending = s.descending
	c.offset = s.offset
	c.limit = s.limit
	return c
}

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
func (s *Search) AddExpr(key, value string) *Search {
	val := parseOp(strings.TrimSpace(value))
	if s == nil {
		s = new(Search)
	}
	s.mx.Lock()
	defer s.mx.Unlock()
	s.addExpValue(key, val)
	return s
}

func (s *Search) addExpValue(key string, val expValue) {
	if key == "" {
		s.addSearch(val)
	} else if s.mvals == nil {
		s.mvals = expMetaValues{key: {val}}
	} else {
		s.mvals[key] = append(s.mvals[key], val)
	}
}

func (s *Search) addSearch(val expValue) {
	if val.negate {
		val.op = val.op.negate()
		val.negate = false
	}
	switch val.op {
	case cmpDefault:
		val.op = cmpContains
	case cmpNotDefault:
		val.op = cmpContains
		val.negate = true
	case cmpNotEqual, cmpNoPrefix, cmpNoSuffix, cmpNotContains:
		val.op = val.op.negate()
		val.negate = true
	}
	s.search = append(s.search, val)
}

func parseOp(s string) expValue {
	if s == "" {
		return expValue{value: s, op: cmpDefault, negate: false}
	}
	if s[0] == '\\' {
		return expValue{value: s[1:], op: cmpDefault, negate: false}
	}
	negate := false
	if s[0] == '!' {
		negate = true
		s = s[1:]
	}
	if s == "" {
		return expValue{value: s, op: cmpDefault, negate: negate}
	}
	if s[0] == '\\' {
		return expValue{value: s[1:], op: cmpDefault, negate: negate}
	}
	switch s[0] {
	case ':':
		return expValue{value: s[1:], op: cmpDefault, negate: negate}
	case '=':
		return expValue{value: s[1:], op: cmpEqual, negate: negate}
	case '>':
		return expValue{value: s[1:], op: cmpPrefix, negate: negate}
	case '<':
		return expValue{value: s[1:], op: cmpSuffix, negate: negate}
	case '~':
		return expValue{value: s[1:], op: cmpContains, negate: negate}
	}
	return expValue{value: s, op: cmpDefault, negate: negate}
}

// SetNegate changes the search to reverse its selection.
func (s *Search) SetNegate() *Search {
	s = createIfNeeded(s)
	s.mx.Lock()
	defer s.mx.Unlock()
	s.negate = true
	return s
}

// AddPreMatch adds the pre-selection predicate.
func (s *Search) AddPreMatch(preMatch MetaMatchFunc) *Search {
	s = createIfNeeded(s)
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
	s = createIfNeeded(s)
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
	s = createIfNeeded(s)
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
	s = createIfNeeded(s)
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
// is calculated via metadata enrichments.
func (s *Search) EnrichNeeded() bool {
	if s == nil {
		return false
	}
	s.mx.RLock()
	defer s.mx.RUnlock()
	for key := range s.mvals {
		if meta.IsComputed(key) {
			return true
		}
	}
	return meta.IsComputed(s.order)
}

// RetrieveAndCompileMatch queries the search index and returns a predicate
// for its results and returns a matching predicate.
func (s *Search) RetrieveAndCompileMatch(searcher Searcher) (RetrievePredicate, MetaMatchFunc) {
	if s == nil {
		return alwaysIncluded, matchAlways
	}
	s = s.Clone()
	match := s.compileMatch() // Match might add some searches
	var pred RetrievePredicate
	if searcher != nil {
		pred = s.retrieveIndex(searcher)
	}

	if pred == nil {
		if match == nil {
			if s.negate {
				return neverIncluded, matchNever
			}
			return alwaysIncluded, matchAlways
		}
		return alwaysIncluded, match
	}
	if match == nil {
		return pred, matchAlways
	}
	return pred, match
}

// retrieveIndex and return a predicate to ask for results.
func (s *Search) retrieveIndex(searcher Searcher) RetrievePredicate {
	if len(s.search) == 0 {
		return nil
	}
	normCalls, plainCalls, negCalls := prepareRetrieveCalls(searcher, s.search)
	if hasConflictingCalls(normCalls, plainCalls, negCalls) {
		return s.neverWithNegate()
	}

	negate := s.negate
	positives := retrievePositives(normCalls, plainCalls)
	if positives == nil {
		// No positive search for words, must contain only words for a negative search.
		// Otherwise len(search) == 0 (see above)
		negatives := retrieveNegatives(negCalls)
		return func(zid id.Zid) bool { return negatives.Contains(zid) == negate }
	}
	if len(positives) == 0 {
		// Positive search didn't found anything. We can omit the negative search.
		return s.neverWithNegate()
	}
	if len(negCalls) == 0 {
		// Positive search found something, but there is no negative search.
		return func(zid id.Zid) bool { return positives.Contains(zid) != negate }
	}
	negatives := retrieveNegatives(negCalls)
	return func(zid id.Zid) bool {
		return (positives.Contains(zid) && !negatives.Contains(zid)) != negate
	}
}

func (s *Search) neverWithNegate() RetrievePredicate {
	if s.negate {
		return alwaysIncluded
	}
	return neverIncluded
}

// compileMatch returns a function to match metadata based on select specification.
func (s *Search) compileMatch() MetaMatchFunc {
	compMeta := s.compileMeta()
	preMatch := s.preMatch
	if compMeta == nil {
		if preMatch == nil {
			return nil
		}
		return preMatch
	}
	if s.negate {
		if preMatch == nil {
			return func(m *meta.Meta) bool { return !compMeta(m) }
		}
		return func(m *meta.Meta) bool { return preMatch(m) && !compMeta(m) }
	}
	if preMatch == nil {
		return compMeta
	}
	return func(m *meta.Meta) bool { return preMatch(m) && compMeta(m) }
}

func matchAlways(*meta.Meta) bool { return true }
func matchNever(*meta.Meta) bool  { return false }

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
	return s.Limit(metaList)
}

// Limit returns only s.GetLimit() elements of the given list.
func (s *Search) Limit(metaList []*meta.Meta) []*meta.Meta {
	if s == nil {
		return metaList
	}
	if s.limit > 0 && s.limit < len(metaList) {
		return metaList[:s.limit]
	}
	return metaList
}
