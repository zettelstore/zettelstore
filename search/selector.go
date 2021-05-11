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
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place/manager/index"
	"zettelstore.de/z/strfun"
)

func compileFullSearch(selector index.Selector, search []expValue) MetaMatchFunc {
	normSearch := compileNormalizedSearch(selector, search)
	plainSearch := compilePlainSearch(selector, search)
	if normSearch == nil {
		if plainSearch == nil {
			return nil
		}
		return plainSearch
	}
	if plainSearch == nil {
		return normSearch
	}
	return func(m *meta.Meta) bool {
		return normSearch(m) || plainSearch(m)
	}
}

func compileNormalizedSearch(selector index.Selector, search []expValue) MetaMatchFunc {
	var positives, negatives []expValue
	posSet := make(map[string]bool)
	negSet := make(map[string]bool)
	for _, val := range search {
		for _, word := range strfun.NormalizeWords(val.value) {
			if val.negate {
				if _, ok := negSet[word]; !ok {
					negSet[word] = true
					negatives = append(negatives, expValue{
						value:  word,
						op:     val.op,
						negate: true,
					})
				}
			} else {
				if _, ok := posSet[word]; !ok {
					posSet[word] = true
					positives = append(positives, expValue{
						value:  word,
						op:     val.op,
						negate: false,
					})
				}
			}
		}
	}
	return compileSearch(selector, positives, negatives)
}
func compilePlainSearch(selector index.Selector, search []expValue) MetaMatchFunc {
	var positives, negatives []expValue
	for _, val := range search {
		if val.negate {
			negatives = append(negatives, expValue{
				value:  strings.ToLower(strings.TrimSpace(val.value)),
				op:     val.op,
				negate: true,
			})
		} else {
			positives = append(positives, expValue{
				value:  strings.ToLower(strings.TrimSpace(val.value)),
				op:     val.op,
				negate: false,
			})
		}
	}
	return compileSearch(selector, positives, negatives)
}

func compileSearch(selector index.Selector, poss, negs []expValue) MetaMatchFunc {
	if len(poss) == 0 {
		if len(negs) == 0 {
			return nil
		}
		return makeNegOnlySearch(selector, negs)
	}
	if len(negs) == 0 {
		return makePosOnlySearch(selector, poss)
	}
	return makePosNegSearch(selector, poss, negs)
}

func makePosOnlySearch(selector index.Selector, poss []expValue) MetaMatchFunc {
	retrievePos := compileRetrieveZids(selector, poss)
	var ids id.Set
	return func(m *meta.Meta) bool {
		if ids == nil {
			ids = retrievePos()
		}
		_, ok := ids[m.Zid]
		return ok
	}
}

func makeNegOnlySearch(selector index.Selector, negs []expValue) MetaMatchFunc {
	retrieveNeg := compileRetrieveZids(selector, negs)
	var ids id.Set
	return func(m *meta.Meta) bool {
		if ids == nil {
			ids = retrieveNeg()
		}
		_, ok := ids[m.Zid]
		return !ok
	}
}

func makePosNegSearch(selector index.Selector, poss, negs []expValue) MetaMatchFunc {
	retrievePos := compileRetrieveZids(selector, poss)
	retrieveNeg := compileRetrieveZids(selector, negs)
	var ids id.Set
	return func(m *meta.Meta) bool {
		if ids == nil {
			ids = retrievePos()
			ids.Remove(retrieveNeg())
		}
		_, okPos := ids[m.Zid]
		return okPos
	}
}

func compileRetrieveZids(selector index.Selector, values []expValue) func() id.Set {
	selFuncs := make([]selectorFunc, 0, len(values))
	stringVals := make([]string, 0, len(values))
	for _, val := range values {
		selFuncs = append(selFuncs, compileSelectOp(selector, val.op))
		stringVals = append(stringVals, val.value)
	}
	if len(selFuncs) == 0 {
		return func() id.Set { return id.NewSet() }
	}
	if len(selFuncs) == 1 {
		return func() id.Set { return selFuncs[0](stringVals[0]) }
	}
	return func() id.Set {
		result := selFuncs[0](stringVals[0])
		for i, f := range selFuncs[1:] {
			result = result.Intersect(f(stringVals[i+1]))
		}
		return result
	}
}

type selectorFunc func(string) id.Set

func compileSelectOp(selector index.Selector, op compareOp) selectorFunc {
	switch op {
	case cmpDefault, cmpContains:
		return selector.SelectContains
	case cmpEqual:
		return selector.SelectEqual
	case cmpPrefix:
		return selector.SelectPrefix
	case cmpSuffix:
		return selector.SelectSuffix
	default:
		panic(fmt.Sprintf("Unexpected value of comparison operation: %v", op))
	}
}
