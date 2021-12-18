//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package search provides a zettel search.
package search

// This file is about "compiling" a search expression into a function.

import (
	"fmt"
	"strings"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/strfun"
)

func compileIndexSearch(searcher Searcher, search []expValue) (RetrieveFunc, RetrieveFunc) {
	if len(search) == 0 {
		return nil, nil
	}
	posNorm, posPlain, negatives := prepareSearchWords(search)
	return compileRetrievePosZids(searcher, posNorm, posPlain), compileRetrieveNegZids(searcher, negatives)
}

type expValueMap map[string]expValue

func prepareSearchWords(search []expValue) (posNorm, posPlain, negatives []expValue) {
	posNormSet := make(expValueMap, len(search))
	negSet := make(expValueMap, len(search))
	for _, val := range search {
		for _, word := range strfun.NormalizeWords(val.value) {
			if val.negate {
				negSet[word] = expValue{
					value:  word,
					op:     val.op,
					negate: true,
				}
			} else {
				posNormSet[word] = expValue{
					value:  word,
					op:     val.op,
					negate: false,
				}
			}
		}
	}

	posPlainSet := make(expValueMap, len(search))
	for _, val := range search {
		word := strings.ToLower(strings.TrimSpace(val.value))
		if val.negate {
			negSet[word] = expValue{
				value:  word,
				op:     val.op,
				negate: true,
			}
		} else {
			posPlainSet[word] = expValue{
				value:  word,
				op:     val.op,
				negate: false,
			}
		}
	}
	return expValueMaptoList(posNormSet), expValueMaptoList(posPlainSet), expValueMaptoList(negSet)
}

func expValueMaptoList(expSet expValueMap) []expValue {
	result := make([]expValue, 0, len(expSet))
	for _, val := range expSet {
		result = append(result, val)
	}
	return result
}
func compileRetrievePosZids(searcher Searcher, normValues, plainValues []expValue) func() id.Set {
	if len(normValues) == 0 {
		if len(plainValues) == 0 {
			return emptySearchFunc
		}
		return compilePosRetrieveZids(searcher, plainValues)
	}
	if len(plainValues) == 0 {
		return compilePosRetrieveZids(searcher, normValues)
	}
	normRetriever := compilePosRetrieveZids(searcher, normValues)
	plainRetriever := compilePosRetrieveZids(searcher, plainValues)
	return func() id.Set {
		return normRetriever().Add(plainRetriever())
	}
}

func compilePosRetrieveZids(searcher Searcher, values []expValue) func() id.Set {
	selFuncs, stringVals := getSearcherCalls(searcher, values)
	if len(selFuncs) == 1 {
		return makeSimpleSearchFunc(selFuncs[0], stringVals[0])
	}
	return func() id.Set {
		result := selFuncs[0](stringVals[0])
		for i, f := range selFuncs[1:] {
			result = result.Intersect(f(stringVals[i+1]))
		}
		return result
	}
}

func compileRetrieveNegZids(searcher Searcher, values []expValue) func() id.Set {
	if len(values) == 0 {
		return emptySearchFunc
	}

	selFuncs, stringVals := getSearcherCalls(searcher, values)
	if len(selFuncs) == 1 {
		return makeSimpleSearchFunc(selFuncs[0], stringVals[0])
	}
	return func() id.Set {
		result := selFuncs[0](stringVals[0])
		for i, f := range selFuncs[1:] {
			result = result.Add(f(stringVals[i+1]))
		}
		return result
	}
}

type selectorFunc func(string) id.Set

func getSearcherCalls(searcher Searcher, values []expValue) ([]selectorFunc, []string) {
	selFuncs := make([]selectorFunc, 0, len(values))
	stringVals := make([]string, 0, len(values))
	for _, val := range values {
		selFuncs = append(selFuncs, compileSelectOp(searcher, val.op))
		stringVals = append(stringVals, val.value)
	}
	return selFuncs, stringVals
}

func emptySearchFunc() id.Set { return id.Set{} }

func makeSimpleSearchFunc(selFunc selectorFunc, arg string) func() id.Set {
	return func() id.Set { return selFunc(arg) }
}

func compileSelectOp(searcher Searcher, op compareOp) selectorFunc {
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
