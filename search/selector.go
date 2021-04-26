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

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/index"
	"zettelstore.de/z/strfun"
)

func compileSearch(selector index.Selector, search []expValue) MetaMatchFunc {
	poss, negs := normalizeSearchValues(search)
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

func normalizeSearchValues(search []expValue) (positives, negatives []expValue) {
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
	return positives, negatives
}

func makePosOnlySearch(selector index.Selector, poss []expValue) MetaMatchFunc {
	return func(m *meta.Meta) bool {
		ids := retrieveZids(selector, poss)
		_, ok := ids[m.Zid]
		return ok
	}
}

func makeNegOnlySearch(selector index.Selector, negs []expValue) MetaMatchFunc {
	return func(m *meta.Meta) bool {
		ids := retrieveZids(selector, negs)
		_, ok := ids[m.Zid]
		return !ok
	}
}

func makePosNegSearch(selector index.Selector, poss, negs []expValue) MetaMatchFunc {
	return func(m *meta.Meta) bool {
		idsPos := retrieveZids(selector, poss)
		_, okPos := idsPos[m.Zid]
		idsNeg := retrieveZids(selector, negs)
		_, okNeg := idsNeg[m.Zid]
		return okPos && !okNeg
	}
}

func retrieveZids(selector index.Selector, vals []expValue) id.Set {
	var result id.Set
	for i, val := range vals {
		var ids id.Set

		switch val.op {
		case cmpDefault, cmpContains:
			ids = selector.SelectContains(val.value)
		case cmpEqual:
			ids = selector.SelectEqual(val.value)
		case cmpPrefix:
			ids = selector.SelectPrefix(val.value)
		default:
			panic(fmt.Sprintf("Unexpected value of comparison operation: %v (word: %q)", val.op, val.value))
		}
		if i == 0 {
			result = ids
			continue
		}
		result = result.Intersect(ids)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
