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
	terms    []conjTerms

	// Fields to be used for sorting
	order  []sortOrder
	offset int // <= 0: no offset
	limit  int // <= 0: no limit
}

// Compiled is a compiled search, to be used in a Box
type Compiled struct {
	PreMatch MetaMatchFunc // Precondition for Match and Retrieve
	Terms    []CompiledTerm
}

// MetaMatchFunc is a function determine whethe some metadata should be selected or not.
type MetaMatchFunc func(*meta.Meta) bool

func matchAlways(*meta.Meta) bool { return true }
func matchNever(*meta.Meta) bool  { return false }

// CompiledTerm is the preprocessed sequence of conjugated search terms.
type CompiledTerm struct {
	Match    MetaMatchFunc     // Match on metadata
	Retrieve RetrievePredicate // Retrieve from full-text search
}

// RetrievePredicate returns true, if the given Zid is contained in the (full-text) search.
type RetrievePredicate func(id.Zid) bool

type keyExistMap map[string]compareOp
type expMetaValues map[string][]expValue

type conjTerms struct {
	keys   keyExistMap
	mvals  expMetaValues // Expected values for a meta datum
	search []expValue    // Search string
}

func (ct *conjTerms) isEmpty() bool {
	return len(ct.keys) == 0 && len(ct.mvals) == 0 && len(ct.search) == 0
}
func (ct *conjTerms) addKey(key string, op compareOp) {
	if ct.keys == nil {
		ct.keys = map[string]compareOp{key: op}
		return
	}
	if prevOp, found := ct.keys[key]; found {
		if prevOp != op {
			ct.keys[key] = cmpUnknown
		}
		return
	}
	ct.keys[key] = op
}
func (ct *conjTerms) addSearch(val expValue) { ct.search = append(ct.search, val) }

type sortOrder struct {
	key        string
	descending bool
}

func (so *sortOrder) isRandom() bool { return so.key == "" }

func createIfNeeded(s *Search) *Search {
	if s == nil {
		return &Search{
			terms: []conjTerms{{}},
		}
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
	c.terms = make([]conjTerms, len(s.terms))
	for i, term := range s.terms {
		if len(term.keys) > 0 {
			c.terms[i].keys = make(keyExistMap, len(term.keys))
			for k, v := range term.keys {
				c.terms[i].keys[k] = v
			}
		}
		// if len(c.mvals) > 0 {
		c.terms[i].mvals = make(expMetaValues, len(term.mvals))
		for k, v := range term.mvals {
			c.terms[i].mvals[k] = v
		}
		// }
		if len(term.search) > 0 {
			c.terms[i].search = append([]expValue{}, term.search...)
		}
	}
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

func (s *Search) addSearch(val expValue) { s.terms[len(s.terms)-1].addSearch(val) }

func (s *Search) addKey(key string, op compareOp) *Search {
	s = createIfNeeded(s)
	s.terms[len(s.terms)-1].addKey(key, op)
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
	for _, term := range s.terms {
		for key := range term.keys {
			if meta.IsComputed(key) {
				return true
			}
		}
		for key := range term.mvals {
			if meta.IsComputed(key) {
				return true
			}
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
			Terms: []CompiledTerm{{
				Match:    matchAlways,
				Retrieve: alwaysIncluded,
			}}}
	}
	s = s.Clone()

	preMatch := s.preMatch
	if preMatch == nil {
		preMatch = matchAlways
	}
	result := Compiled{PreMatch: preMatch}

	for _, term := range s.terms {
		cTerm := term.retrievAndCompileTerm(searcher)
		if cTerm.Retrieve == nil {
			if cTerm.Match == nil {
				// no restriction on match/retrieve -> all will match
				return Compiled{
					PreMatch: preMatch,
					Terms: []CompiledTerm{{
						Match:    matchAlways,
						Retrieve: alwaysIncluded,
					}}}
			}
			cTerm.Retrieve = alwaysIncluded
		}
		if cTerm.Match == nil {
			cTerm.Match = matchAlways
		}
		result.Terms = append(result.Terms, cTerm)
	}
	return result
}

func (ct *conjTerms) retrievAndCompileTerm(searcher Searcher) CompiledTerm {
	match := ct.compileMeta() // Match might add some searches
	var pred RetrievePredicate
	if searcher != nil {
		pred = ct.retrieveIndex(searcher)
	}
	return CompiledTerm{Match: match, Retrieve: pred}
}

// retrieveIndex and return a predicate to ask for results.
func (ct *conjTerms) retrieveIndex(searcher Searcher) RetrievePredicate {
	if len(ct.search) == 0 {
		return nil
	}
	normCalls, plainCalls, negCalls := prepareRetrieveCalls(searcher, ct.search)
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
