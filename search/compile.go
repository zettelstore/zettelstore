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

func emptySearchFunc() id.Set { return id.Set{} }

func compileIndexSearch(searcher Searcher, search []expValue) (RetrieveFunc, RetrieveFunc) {
	if len(search) == 0 {
		return nil, nil
	}
	posNorm, posPlain, negatives := prepareSearchCalls(searcher, search)
	if hasConflictingCalls(posNorm, posPlain, negatives) {
		return emptySearchFunc, emptySearchFunc
	}
	if isSuperset(posNorm, posPlain) {
		posPlain = nil
	}

	negRetrieveFunc := emptySearchFunc
	if len(negatives) > 0 {
		negRetrieveFunc = func() id.Set {
			var result id.Set
			for val, sf := range negatives {
				result = result.Add(sf(val.s))
			}
			return result

		}
	}
	return compileRetrievePosZids(posNorm, posPlain), negRetrieveFunc
}

func prepareSearchCalls(searcher Searcher, search []expValue) (posNorm, posPlain, negatives searchCallMap) {
	posNorm = make(searchCallMap, len(search))
	negatives = make(searchCallMap, len(search))
	for _, val := range search {
		for _, word := range strfun.NormalizeWords(val.value) {
			sf := getSearchFunc(searcher, val.op)
			if val.negate {
				negatives[searchOp{s: word, op: val.op}] = sf
			} else {
				posNorm[searchOp{s: word, op: val.op}] = sf
			}
		}
	}

	posPlain = make(searchCallMap, len(search))
	for _, val := range search {
		word := strings.ToLower(strings.TrimSpace(val.value))
		sf := getSearchFunc(searcher, val.op)
		if val.negate {
			negatives[searchOp{s: word, op: val.op}] = sf
		} else {
			posPlain[searchOp{s: word, op: val.op}] = sf
		}
	}
	return posNorm, posPlain, negatives
}

func hasConflictingCalls(posNorm, posPlain, negatives searchCallMap) bool {
	for val := range negatives {
		if _, found := posNorm[val]; found {
			return true
		}
		if _, found := posPlain[val]; found {
			return true
		}
	}
	return false
}

func isSuperset(posNorm, posPlain searchCallMap) bool {
	for c := range posPlain {
		if _, found := posNorm[c]; !found {
			return false
		}
	}
	return true
}

type searchResults map[searchOp]id.Set

func compileRetrievePosZids(normCalls, plainCalls searchCallMap) func() id.Set {
	if len(normCalls) == 0 {
		if len(plainCalls) == 0 {
			return emptySearchFunc
		}
		return compilePosRetrieveZids(plainCalls)
	}
	if len(plainCalls) == 0 {
		return compilePosRetrieveZids(normCalls)
	}
	return func() id.Set { return searchPositives(normCalls, plainCalls) }
}

func compilePosRetrieveZids(searchCalls searchCallMap) func() id.Set {
	return func() id.Set {
		var result id.Set
		for c, sf := range searchCalls {
			result = result.IntersectOrSet(sf(c.s))
		}
		return result
	}
}

func searchPositives(normCalls, plainCalls searchCallMap) id.Set {
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
