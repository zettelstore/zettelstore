//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package query provides a query for zettel.
package query

import (
	"math/rand"
	"sort"

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

// Query specifies a mechanism for querying zettel.
type Query struct {
	// Fields to be used for selecting
	preMatch MetaMatchFunc // Match that must be true
	terms    []conjTerms

	// Allow to create predictable randomness
	seed int

	pick int // Randomly pick elements, <= 0: no pick

	// Fields to be used for sorting
	order  []sortOrder
	offset int // <= 0: no offset
	limit  int // <= 0: no limit

	// Execute specification
	actions []string
}

// Compiled is a compiled query, to be used in a Box
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

func createIfNeeded(q *Query) *Query {
	if q == nil {
		return &Query{
			terms: []conjTerms{{}},
		}
	}
	return q
}

// Clone the query value.
func (q *Query) Clone() *Query {
	if q == nil {
		return nil
	}
	c := new(Query)
	c.preMatch = q.preMatch
	c.terms = make([]conjTerms, len(q.terms))
	for i, term := range q.terms {
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
	c.seed = q.seed
	c.pick = q.pick
	if len(q.order) > 0 {
		c.order = append([]sortOrder{}, q.order...)
	}
	c.offset = q.offset
	c.limit = q.limit
	c.actions = q.actions
	return c
}

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

func (q *Query) addSearch(val expValue) { q.terms[len(q.terms)-1].addSearch(val) }

func (q *Query) addKey(key string, op compareOp) *Query {
	q = createIfNeeded(q)
	q.terms[len(q.terms)-1].addKey(key, op)
	return q
}

// SetPreMatch sets the pre-selection predicate.
func (q *Query) SetPreMatch(preMatch MetaMatchFunc) *Query {
	q = createIfNeeded(q)
	if q.preMatch != nil {
		panic("search PreMatch already set")
	}
	q.preMatch = preMatch
	return q
}

// SetSeed sets a seed value.
func (q *Query) SetSeed(seed int) *Query {
	q = createIfNeeded(q)
	q.seed = seed
	return q
}

// GetSeed returns the seed value if one was set.
func (q *Query) GetSeed() (int, bool) {
	if q == nil {
		return 0, false
	}
	return q.seed, q.seed > 0
}

// SetDeterministic signals that the result should be the same if the seed is the same.
func (q *Query) SetDeterministic() *Query {
	q = createIfNeeded(q)
	if q.seed <= 0 {
		q.seed = int(rand.Intn(10000) + 1)
	}
	return q
}

func (q *Query) isDeterministic() bool { return q.seed > 0 }

// SetLimit sets the given limit of the query object.
func (q *Query) SetLimit(limit int) *Query {
	q = createIfNeeded(q)
	if limit < 0 {
		limit = 0
	}
	q.limit = limit
	return q
}

// GetLimit returns the current offset value.
func (q *Query) GetLimit() int {
	if q == nil {
		return 0
	}
	return q.limit
}

// Actions returns the slice of action specifications
func (q *Query) Actions() []string {
	if q == nil {
		return nil
	}
	return q.actions
}

// RemoveActions will remove the action part of a query.
func (q *Query) RemoveActions() {
	if q != nil {
		q.actions = nil
	}
}

// EnrichNeeded returns true, if the query references a metadata key that
// is calculated via metadata enrichments.
func (q *Query) EnrichNeeded() bool {
	if q == nil {
		return false
	}
	if len(q.actions) > 0 {
		// Unknown, what an action will use. Example: RSS needs api.KeyPublished.
		return true
	}
	for _, term := range q.terms {
		for key := range term.keys {
			if meta.IsProperty(key) {
				return true
			}
		}
		for key := range term.mvals {
			if meta.IsProperty(key) {
				return true
			}
		}
	}
	for _, o := range q.order {
		if meta.IsProperty(o.key) {
			return true
		}
	}
	return false
}

// RetrieveAndCompile queries the search index and returns a predicate
// for its results and returns a matching predicate.
func (q *Query) RetrieveAndCompile(searcher Searcher) Compiled {
	if q == nil {
		return Compiled{
			PreMatch: matchAlways,
			Terms: []CompiledTerm{{
				Match:    matchAlways,
				Retrieve: alwaysIncluded,
			}}}
	}
	q = q.Clone()

	preMatch := q.preMatch
	if preMatch == nil {
		preMatch = matchAlways
	}
	result := Compiled{PreMatch: preMatch}

	for _, term := range q.terms {
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

// AfterSearch applies all terms to the metadata list that was searched.
//
// This includes sorting, offset, limit, and picking.
func (q *Query) AfterSearch(metaList []*meta.Meta) []*meta.Meta {
	if len(metaList) == 0 {
		return metaList
	}

	if q == nil {
		return sortMetaByZid(metaList)
	}

	metaList = q.doPick(metaList)

	if len(q.order) == 0 {
		if q.pick <= 0 {
			metaList = sortMetaByZid(metaList)
		}
	} else if q.order[0].isRandom() {
		metaList = q.sortRandomly(metaList)
	} else {
		sort.Slice(metaList, createSortFunc(q.order, metaList))
	}

	if q.offset > 0 {
		if q.offset > len(metaList) {
			return nil
		}
		metaList = metaList[q.offset:]
	}
	return q.Limit(metaList)
}

func (q *Query) doPick(metaList []*meta.Meta) []*meta.Meta {
	pick := q.pick
	if pick <= 0 {
		return metaList
	}
	if limit := q.limit; limit > 0 && limit < pick {
		pick = limit
	}
	if pick >= len(metaList) {
		return q.doRandom(metaList)
	}
	return q.doPickN(metaList, pick)
}
func (q *Query) doPickN(metaList []*meta.Meta, pick int) []*meta.Meta {
	if q.isDeterministic() {
		metaList = sortMetaByZid(metaList)
	}
	rnd := q.newRandom()
	result := make([]*meta.Meta, pick)
	for i := 0; i < pick; i++ {
		n := rnd.Intn(pick - i)
		result[i] = metaList[n]
		metaList[n] = metaList[len(metaList)-i-1]
		metaList[len(metaList)-i-1] = nil
	}
	return result
}

func (q *Query) sortRandomly(metaList []*meta.Meta) []*meta.Meta {
	// Optimization: RANDOM LIMIT n, where n < len(metaList) is essentially a PICK n.
	if limit := q.limit; limit > 0 && limit < len(metaList) {
		return q.doPickN(metaList, limit)
	}
	return q.doRandom(metaList)
}
func (q *Query) doRandom(metaList []*meta.Meta) []*meta.Meta {
	if q.isDeterministic() {
		metaList = sortMetaByZid(metaList)
	}
	rnd := q.newRandom()
	rnd.Shuffle(
		len(metaList),
		func(i, j int) { metaList[i], metaList[j] = metaList[j], metaList[i] },
	)
	return metaList
}

func (q *Query) newRandom() *rand.Rand {
	if q.seed <= 0 {
		return rand.New(rand.NewSource(int64(rand.Intn(10000) + 10001)))
	}
	return rand.New(rand.NewSource(int64(q.seed)))
}

// Limit returns only s.GetLimit() elements of the given list.
func (q *Query) Limit(metaList []*meta.Meta) []*meta.Meta {
	if q == nil {
		return metaList
	}
	if limit := q.limit; limit > 0 && limit < len(metaList) {
		return metaList[:limit]
	}
	return metaList
}

func sortMetaByZid(metaList []*meta.Meta) []*meta.Meta {
	sort.Slice(metaList, func(i, j int) bool { return metaList[i].Zid > metaList[j].Zid })
	return metaList
}
