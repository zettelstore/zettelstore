//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package search

// This file is about "compiling" a search expression into a function.

import (
	"fmt"
	"strings"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/strfun"
)

type searchOp struct {
	s  string
	op compareOp
}
type searchFunc func(string) id.Set
type searchCallMap map[searchOp]searchFunc

func alwaysIncluded(id.Zid) bool { return true }
func neverIncluded(id.Zid) bool  { return false }

func compileIndexSearch(searcher Searcher, search []expValue) RetrievePredicate {
	if len(search) == 0 {
		return alwaysIncluded
	}
	normCalls, plainCalls, negCalls := prepareSearchCalls(searcher, search)
	if hasConflictingCalls(normCalls, plainCalls, negCalls) {
		return neverIncluded
	}

	positives := searchPositives(normCalls, plainCalls)
	if positives == nil {
		// No positive search for words, must contain only words for a negative search.
		// Otherwise len(search) == 0 (see above)
		negatives := searchNegatives(negCalls)
		return func(zid id.Zid) bool { return !negatives.Contains(zid) }
	}
	if len(positives) == 0 {
		// Positive search didn't found anything. We can omit the negative search.
		return neverIncluded
	}
	if len(negCalls) == 0 {
		// Positive search found something, but there is no negative search.
		return func(zid id.Zid) bool { return positives.Contains(zid) }
	}
	negatives := searchNegatives(negCalls)
	return func(zid id.Zid) bool { return positives.Contains(zid) && !negatives.Contains(zid) }
}

func prepareSearchCalls(searcher Searcher, search []expValue) (normCalls, plainCalls, negCalls searchCallMap) {
	normCalls = make(searchCallMap, len(search))
	negCalls = make(searchCallMap, len(search))
	for _, val := range search {
		for _, word := range strfun.NormalizeWords(val.value) {
			sf := getSearchFunc(searcher, val.op)
			if val.negate {
				negCalls[searchOp{s: word, op: val.op}] = sf
			} else {
				normCalls[searchOp{s: word, op: val.op}] = sf
			}
		}
	}

	plainCalls = make(searchCallMap, len(search))
	for _, val := range search {
		word := strings.ToLower(strings.TrimSpace(val.value))
		sf := getSearchFunc(searcher, val.op)
		if val.negate {
			negCalls[searchOp{s: word, op: val.op}] = sf
		} else {
			plainCalls[searchOp{s: word, op: val.op}] = sf
		}
	}
	return normCalls, plainCalls, negCalls
}

func hasConflictingCalls(normCalls, plainCalls, negCalls searchCallMap) bool {
	for val := range negCalls {
		if _, found := normCalls[val]; found {
			return true
		}
		if _, found := plainCalls[val]; found {
			return true
		}
	}
	return false
}

func searchPositives(normCalls, plainCalls searchCallMap) id.Set {
	if isSuperset(normCalls, plainCalls) {
		var normResult id.Set
		for c, sf := range normCalls {
			normResult = normResult.IntersectOrSet(sf(c.s))
		}
		return normResult
	}

	type searchResults map[searchOp]id.Set
	var cache searchResults
	var plainResult id.Set
	for c, sf := range plainCalls {
		result := sf(c.s)
		if _, found := normCalls[c]; found {
			if cache == nil {
				cache = make(searchResults)
			}
			cache[c] = result
		}
		plainResult = plainResult.IntersectOrSet(result)
	}
	var normResult id.Set
	for c, sf := range normCalls {
		if cache != nil {
			if result, found := cache[c]; found {
				normResult = normResult.IntersectOrSet(result)
				continue
			}
		}
		normResult = normResult.IntersectOrSet(sf(c.s))
	}
	return normResult.Add(plainResult)
}

func isSuperset(normCalls, plainCalls searchCallMap) bool {
	for c := range plainCalls {
		if _, found := normCalls[c]; !found {
			return false
		}
	}
	return true
}

func searchNegatives(negCalls searchCallMap) id.Set {
	var negatives id.Set
	for val, sf := range negCalls {
		negatives = negatives.Add(sf(val.s))
	}
	return negatives
}

func getSearchFunc(searcher Searcher, op compareOp) searchFunc {
	switch op {
	case cmpDefault, cmpContains:
		return searcher.SearchContains
	case cmpEqual:
		return searcher.SearchEqual
	case cmpPrefix:
		return searcher.SearchPrefix
	case cmpSuffix:
		return searcher.SearchSuffix
	default:
		panic(fmt.Sprintf("Unexpected value of comparison operation: %v", op))
	}
}
