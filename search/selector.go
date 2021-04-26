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
	var ids id.Set
	return func(m *meta.Meta) bool {
		if ids == nil {
			ids = retrieveZids(selector, poss)
		}
		_, ok := ids[m.Zid]
		return ok
	}
}

func makeNegOnlySearch(selector index.Selector, negs []expValue) MetaMatchFunc {
	var ids id.Set
	return func(m *meta.Meta) bool {
		if ids == nil {
			ids = retrieveZids(selector, negs)
		}
		_, ok := ids[m.Zid]
		return !ok
	}
}

func makePosNegSearch(selector index.Selector, poss, negs []expValue) MetaMatchFunc {
	var idsPos id.Set
	return func(m *meta.Meta) bool {
		if idsPos == nil {
			idsPos = retrieveZids(selector, poss)
			idsNeg := retrieveZids(selector, negs)
			idsPos.Remove(idsNeg)
		}
		_, okPos := idsPos[m.Zid]
		return okPos
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
	return result
}
