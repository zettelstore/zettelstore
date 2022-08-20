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
	"sync"

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

// Search specifies a mechanism for selecting zettel.
type Search struct {
	mx sync.RWMutex // Protects other attributes

	// Fields to be used for selecting
	preMatch MetaMatchFunc // Match that must be true
	keyExist keyExistMap
	mvals    expMetaValues // Expected values for a meta datum
	search   []expValue    // Search string
	negate   bool          // Negate the result of the whole selecting process

	// Fields to be used for sorting
	order  []sortOrder
	offset int // <= 0: no offset
	limit  int // <= 0: no limit
}

// Compiled is a compiled search, to be used in a Box
type Compiled struct {
	PreMatch MetaMatchFunc     // Precondition for Match and Retrieve
	Match    MetaMatchFunc     // Match on metadata
	Retrieve RetrievePredicate // Retrieve from full-text search
}

// MetaMatchFunc is a function determine whethe some metadata should be selected or not.
type MetaMatchFunc func(*meta.Meta) bool

func matchAlways(*meta.Meta) bool { return true }
func matchNever(*meta.Meta) bool  { return false }

// RetrievePredicate returns true, if the given Zid is contained in the (full-text) search.
type RetrievePredicate func(id.Zid) bool

type keyExistMap map[string]compareOp
type expMetaValues map[string][]expValue

type sortOrder struct {
	key        string
	descending bool
}

func (so *sortOrder) isRandom() bool { return so.key == "" }

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
	if len(s.keyExist) > 0 {
		c.keyExist = make(keyExistMap, len(s.keyExist))
		for k, v := range s.keyExist {
			c.keyExist[k] = v
		}
	}
	// if len(c.mvals) > 0 {
	c.mvals = make(expMetaValues, len(s.mvals))
	for k, v := range s.mvals {
		c.mvals[k] = v
	}
	// }
	if len(s.search) > 0 {
		c.search = append([]expValue{}, s.search...)
	}
	c.negate = s.negate
	if len(s.order) > 0 {
		c.order = append([]sortOrder{}, s.order...)
	}
	c.offset = s.offset
	c.limit = s.limit
	return c
}

// RandomOrder is a pseudo metadata key that selects a random order.
const RandomOrder = "_random"

type compareOp uint8

const (
	cmpUnknown compareOp = iota
	cmpExist
	cmpNotExist
	cmpHas
	cmpHasNot
	cmpPrefix
	cmpNoPrefix
	cmpSuffix
	cmpNoSuffix
	cmpMatch
	cmpNoMatch
)

var negateMap = map[compareOp]compareOp{
	cmpUnknown:  cmpUnknown,
	cmpExist:    cmpNotExist,
	cmpHas:      cmpHasNot,
	cmpHasNot:   cmpHas,
	cmpPrefix:   cmpNoPrefix,
	cmpNoPrefix: cmpPrefix,
	cmpSuffix:   cmpNoSuffix,
	cmpNoSuffix: cmpSuffix,
	cmpMatch:    cmpNoMatch,
	cmpNoMatch:  cmpMatch,
}

func (op compareOp) negate() compareOp { return negateMap[op] }

var negativeMap = map[compareOp]bool{
	cmpNotExist: true,
	cmpHasNot:   true,
	cmpNoPrefix: true,
	cmpNoSuffix: true,
	cmpNoMatch:  true,
}

func (op compareOp) isNegated() bool { return negativeMap[op] }

type expValue struct {
	value string
	op    compareOp
}

func (s *Search) addSearch(val expValue) { s.search = append(s.search, val) }

func (s *Search) addKeyExist(key string, op compareOp) *Search {
	s = createIfNeeded(s)
	if s.keyExist == nil {
		s.keyExist = map[string]compareOp{key: op}
		return s
	}
	if prevOp, found := s.keyExist[key]; found {
		if prevOp != op {
			s.keyExist[key] = cmpUnknown
		}
		return s
	}
	s.keyExist[key] = op
	return s
}

// SetPreMatch sets the pre-selection predicate.
func (s *Search) SetPreMatch(preMatch MetaMatchFunc) *Search {
	s = createIfNeeded(s)
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.preMatch != nil {
		panic("search PreMatch already set")
	}
	s.preMatch = preMatch
	return s
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
	for key := range s.keyExist {
		if meta.IsComputed(key) {
			return true
		}
	}
	for key := range s.mvals {
		if meta.IsComputed(key) {
			return true
		}
	}
	for _, o := range s.order {
		if meta.IsComputed(o.key) {
			return true
		}
	}
	return false
}

// RetrieveAndCompile queries the search index and returns a predicate
// for its results and returns a matching predicate.
func (s *Search) RetrieveAndCompile(searcher Searcher) Compiled {
	if s == nil {
		return Compiled{
			PreMatch: matchAlways,
			Match:    matchAlways,
			Retrieve: alwaysIncluded,
		}
	}
	s = s.Clone()

	preMatch := s.preMatch
	if preMatch == nil {
		preMatch = matchAlways
	}
	match := s.compileMeta() // Match might add some searches
	if match != nil && s.negate {
		matchO := match
		match = func(m *meta.Meta) bool { return !matchO(m) }
	}
	var pred RetrievePredicate
	if searcher != nil {
		pred = s.retrieveIndex(searcher)
		if pred != nil && s.negate {
			pred0 := pred
			pred = func(zid id.Zid) bool { return !pred0(zid) }
		}
	}

	if pred == nil {
		if match == nil {
			if s.negate {
				return Compiled{
					PreMatch: matchNever,
					Match:    matchNever,
					Retrieve: neverIncluded,
				}
			}
			return Compiled{
				PreMatch: preMatch,
				Match:    matchAlways,
				Retrieve: alwaysIncluded,
			}
		}
		return Compiled{
			PreMatch: preMatch,
			Match:    match,
			Retrieve: alwaysIncluded,
		}
	}
	if match == nil {
		return Compiled{
			PreMatch: preMatch,
			Match:    matchAlways,
			Retrieve: pred,
		}
	}
	return Compiled{
		PreMatch: preMatch,
		Match:    match,
		Retrieve: pred,
	}
}

// retrieveIndex and return a predicate to ask for results.
func (s *Search) retrieveIndex(searcher Searcher) RetrievePredicate {
	if len(s.search) == 0 {
		return nil
	}
	normCalls, plainCalls, negCalls := prepareRetrieveCalls(searcher, s.search)
	if hasConflictingCalls(normCalls, plainCalls, negCalls) {
		return neverIncluded
	}

	positives := retrievePositives(normCalls, plainCalls)
	if positives == nil {
		// No positive search for words, must contain only words for a negative search.
		// Otherwise len(search) == 0 (see above)
		negatives := retrieveNegatives(negCalls)
		return func(zid id.Zid) bool { return !negatives.Contains(zid) }
	}
	if len(positives) == 0 {
		// Positive search didn't found anything. We can omit the negative search.
		return neverIncluded
	}
	if len(negCalls) == 0 {
		// Positive search found something, but there is no negative search.
		return positives.Contains
	}
	negatives := retrieveNegatives(negCalls)
	return func(zid id.Zid) bool {
		return positives.Contains(zid) && !negatives.Contains(zid)
	}
}

// Sort applies the sorter to the slice of meta data.
func (s *Search) Sort(metaList []*meta.Meta) []*meta.Meta {
	if len(metaList) == 0 {
		return metaList
	}

	if s == nil || len(s.order) == 0 {
		sort.Slice(metaList, func(i, j int) bool { return metaList[i].Zid > metaList[j].Zid })
		if s == nil {
			return metaList
		}
	} else if s.order[0].isRandom() {
		rand.Shuffle(len(metaList), func(i, j int) {
			metaList[i], metaList[j] = metaList[j], metaList[i]
		})
	} else {
		sort.Slice(metaList, createSortFunc(s.order, metaList))
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
