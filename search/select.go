//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package search

import (
	"fmt"
	"strings"

	"zettelstore.de/z/domain/meta"
)

type matchValueFunc func(value string) bool

func matchValueNever(string) bool  { return false }
func matchValueAlways(string) bool { return true }

type matchSpec struct {
	key   string
	match matchValueFunc
}

// compileMeta calculates a selection func based on the given select criteria.
func (s *Search) compileMeta() MetaMatchFunc {
	posSpecs, negSpecs, nomatch := s.createSelectSpecs()
	if len(posSpecs) > 0 || len(negSpecs) > 0 || len(nomatch) > 0 {
		return makeSearchMetaMatchFunc(posSpecs, negSpecs, nomatch)
	}
	return nil
}

func (s *Search) createSelectSpecs() (posSpecs, negSpecs []matchSpec, nomatch []string) {
	posSpecs = make([]matchSpec, 0, len(s.mvals))
	negSpecs = make([]matchSpec, 0, len(s.mvals))
	for key, values := range s.mvals {
		if !meta.KeyIsValid(key) {
			continue
		}
		if always, never := countEmptyValues(values); always+never > 0 {
			if never == 0 {
				posSpecs = append(posSpecs, matchSpec{key, matchValueAlways})
				continue
			}
			if always == 0 {
				negSpecs = append(negSpecs, matchSpec{key, nil})
				continue
			}
			// value must match always AND never, at the same time. This results in a no-match.
			nomatch = append(nomatch, key)
			continue
		}
		posMatch, negMatch := createPosNegMatchFunc(key, values, s.addSearch)
		if posMatch != nil {
			posSpecs = append(posSpecs, matchSpec{key, posMatch})
		}
		if negMatch != nil {
			negSpecs = append(negSpecs, matchSpec{key, negMatch})
		}
	}
	return posSpecs, negSpecs, nomatch
}

func countEmptyValues(values []expValue) (always, never int) {
	for _, v := range values {
		if v.value == "" {
			if v.op.isNegated() {
				never++
			} else {
				always++
			}
		}
	}
	return always, never
}

type addSearchFunc func(val expValue)

func createPosNegMatchFunc(key string, values []expValue, addSearch addSearchFunc) (posMatch, negMatch matchValueFunc) {
	posValues := make([]expValue, 0, len(values))
	negValues := make([]expValue, 0, len(values))
	for _, val := range values {
		if val.op.isNegated() {
			negValues = append(negValues, val)
		} else {
			posValues = append(posValues, val)
		}
	}
	return createMatchFunc(key, posValues, addSearch), createMatchFunc(key, negValues, addSearch)
}

func createMatchFunc(key string, values []expValue, addSearch addSearchFunc) matchValueFunc {
	if len(values) == 0 {
		return nil
	}
	switch meta.Type(key) {
	case meta.TypeCredential:
		return matchValueNever
	case meta.TypeID, meta.TypeTimestamp: // ID and timestamp use the same layout
		return createMatchIDFunc(values, addSearch)
	case meta.TypeIDSet:
		return createMatchIDSetFunc(values, addSearch)
	case meta.TypeTagSet:
		return createMatchTagSetFunc(values, addSearch)
	case meta.TypeWord:
		return createMatchWordFunc(values, addSearch)
	case meta.TypeWordSet:
		return createMatchWordSetFunc(values, addSearch)
	}
	return createMatchStringFunc(values, addSearch)
}

func createMatchIDFunc(values []expValue, addSearch addSearchFunc) matchValueFunc {
	preds := valuesToStringPredicates(values, cmpPrefix, addSearch)
	return func(value string) bool {
		for _, pred := range preds {
			if !pred(value) {
				return false
			}
		}
		return true
	}
}

func createMatchIDSetFunc(values []expValue, addSearch addSearchFunc) matchValueFunc {
	predList := valuesToStringSetPredicates(preprocessSet(values), cmpPrefix, addSearch)
	return func(value string) bool {
		ids := meta.ListFromValue(value)
		for _, preds := range predList {
			for _, pred := range preds {
				if !pred(ids) {
					return false
				}
			}
		}
		return true
	}
}

func createMatchTagSetFunc(values []expValue, addSearch addSearchFunc) matchValueFunc {
	predList := valuesToStringSetPredicates(processTagSet(preprocessSet(sliceToLower(values))), cmpEqual, addSearch)
	return func(value string) bool {
		tags := meta.ListFromValue(value)
		// Remove leading '#' from each tag
		for i, tag := range tags {
			tags[i] = meta.CleanTag(tag)
		}
		for _, preds := range predList {
			for _, pred := range preds {
				if !pred(tags) {
					return false
				}
			}
		}
		return true
	}
}

func processTagSet(valueSet [][]expValue) [][]expValue {
	result := make([][]expValue, len(valueSet))
	for i, values := range valueSet {
		tags := make([]expValue, len(values))
		for j, val := range values {
			if tval := val.value; tval != "" && tval[0] == '#' {
				tval = meta.CleanTag(tval)
				tags[j] = expValue{value: tval, op: val.op}
			} else {
				tags[j] = expValue{value: tval, op: val.op}
			}
		}
		result[i] = tags
	}
	return result
}

func createMatchWordFunc(values []expValue, addSearch addSearchFunc) matchValueFunc {
	preds := valuesToStringPredicates(sliceToLower(values), cmpEqual, addSearch)
	return func(value string) bool {
		value = strings.ToLower(value)
		for _, pred := range preds {
			if !pred(value) {
				return false
			}
		}
		return true
	}
}

func createMatchWordSetFunc(values []expValue, addSearch addSearchFunc) matchValueFunc {
	predsList := valuesToStringSetPredicates(preprocessSet(sliceToLower(values)), cmpEqual, addSearch)
	return func(value string) bool {
		words := meta.ListFromValue(value)
		for _, preds := range predsList {
			for _, pred := range preds {
				if !pred(words) {
					return false
				}
			}
		}
		return true
	}
}

func createMatchStringFunc(values []expValue, addSearch addSearchFunc) matchValueFunc {
	preds := valuesToStringPredicates(sliceToLower(values), cmpContains, addSearch)
	return func(value string) bool {
		value = strings.ToLower(value)
		for _, pred := range preds {
			if !pred(value) {
				return false
			}
		}
		return true
	}
}

func sliceToLower(sl []expValue) []expValue {
	result := make([]expValue, 0, len(sl))
	for _, s := range sl {
		result = append(result, expValue{
			value: strings.ToLower(s.value),
			op:    s.op,
		})
	}
	return result
}

func preprocessSet(set []expValue) [][]expValue {
	result := make([][]expValue, 0, len(set))
	for _, elem := range set {
		splitElems := strings.Split(elem.value, ",")
		valueElems := make([]expValue, 0, len(splitElems))
		for _, se := range splitElems {
			e := strings.TrimSpace(se)
			if len(e) > 0 {
				valueElems = append(valueElems, expValue{value: e, op: elem.op})
			}
		}
		if len(valueElems) > 0 {
			result = append(result, valueElems)
		}
	}
	return result
}

type stringPredicate func(string) bool

func valuesToStringPredicates(values []expValue, defOp compareOp, addSearch addSearchFunc) []stringPredicate {
	result := make([]stringPredicate, len(values))
	for i, v := range values {
		opVal := v.value // loop variable is used in closure --> save needed value
		op := v.op
		switch op {
		case cmpEqual:
			addSearch(v) // addSearch only for positive selections
			result[i] = func(metaVal string) bool { return metaVal == opVal }
		case cmpNotEqual:
			result[i] = func(metaVal string) bool { return metaVal != opVal }
		case cmpPrefix:
			addSearch(v)
			result[i] = func(metaVal string) bool { return strings.HasPrefix(metaVal, opVal) }
		case cmpNoPrefix:
			result[i] = func(metaVal string) bool { return !strings.HasPrefix(metaVal, opVal) }
		case cmpSuffix:
			addSearch(v)
			result[i] = func(metaVal string) bool { return strings.HasSuffix(metaVal, opVal) }
		case cmpNoSuffix:
			result[i] = func(metaVal string) bool { return !strings.HasSuffix(metaVal, opVal) }
		case cmpContains:
			addSearch(v)
			result[i] = func(metaVal string) bool { return strings.Contains(metaVal, opVal) }
		case cmpNotContains:
			result[i] = func(metaVal string) bool { return !strings.Contains(metaVal, opVal) }
		default:
			panic(fmt.Sprintf("Unknown compare operation %d/%d with value %q", op, v.op, opVal))
		}
	}
	return result
}

type stringSetPredicate func(value []string) bool

func valuesToStringSetPredicates(values [][]expValue, defOp compareOp, addSearch addSearchFunc) [][]stringSetPredicate {
	result := make([][]stringSetPredicate, len(values))
	for i, val := range values {
		elemPreds := make([]stringSetPredicate, len(val))
		for j, v := range val {
			opVal := v.value // loop variable is used in closure --> save needed value
			op := v.op
			switch op {
			case cmpEqual:
				addSearch(v) // addSearch only for positive selections
				elemPreds[j] = makeStringSetPredicate(opVal, stringEqual, true)
			case cmpNotEqual:
				elemPreds[j] = makeStringSetPredicate(opVal, stringEqual, false)
			case cmpPrefix:
				addSearch(v)
				elemPreds[j] = makeStringSetPredicate(opVal, strings.HasPrefix, true)
			case cmpNoPrefix:
				elemPreds[j] = makeStringSetPredicate(opVal, strings.HasPrefix, false)
			case cmpSuffix:
				addSearch(v)
				elemPreds[j] = makeStringSetPredicate(opVal, strings.HasSuffix, true)
			case cmpNoSuffix:
				elemPreds[j] = makeStringSetPredicate(opVal, strings.HasSuffix, false)
			case cmpContains:
				addSearch(v)
				elemPreds[j] = makeStringSetPredicate(opVal, strings.Contains, true)
			case cmpNotContains:
				elemPreds[j] = makeStringSetPredicate(opVal, strings.Contains, false)
			default:
				panic(fmt.Sprintf("Unknown compare operation %d/%d with value %q", op, v.op, opVal))
			}
		}
		result[i] = elemPreds
	}
	return result
}

func stringEqual(val1, val2 string) bool { return val1 == val2 }

type compareStringFunc func(val1, val2 string) bool

func makeStringSetPredicate(neededValue string, compare compareStringFunc, foundResult bool) stringSetPredicate {
	return func(metaVals []string) bool {
		for _, metaVal := range metaVals {
			if compare(metaVal, neededValue) {
				return foundResult
			}
		}
		return !foundResult
	}
}

func makeSearchMetaMatchFunc(posSpecs, negSpecs []matchSpec, nomatch []string) MetaMatchFunc {
	if len(nomatch) == 0 {
		// Optimize for simple cases: only negative or only positive matching

		if len(posSpecs) == 0 {
			return func(m *meta.Meta) bool { return matchMetaNegSpecs(m, negSpecs) }
		}
		if len(negSpecs) == 0 {
			return func(m *meta.Meta) bool { return matchMetaPosSpecs(m, posSpecs) }
		}
	}
	return func(m *meta.Meta) bool {
		return matchMetaNoMatch(m, nomatch) &&
			matchMetaPosSpecs(m, posSpecs) &&
			matchMetaNegSpecs(m, negSpecs)
	}
}

func matchMetaNoMatch(m *meta.Meta, nomatch []string) bool {
	for _, key := range nomatch {
		if _, ok := m.Get(key); ok {
			return false
		}
	}
	return true
}
func matchMetaPosSpecs(m *meta.Meta, posSpecs []matchSpec) bool {
	for _, s := range posSpecs {
		if value, ok := m.Get(s.key); !ok || !s.match(value) {
			return false
		}
	}
	return true
}
func matchMetaNegSpecs(m *meta.Meta, negSpecs []matchSpec) bool {
	for _, s := range negSpecs {
		if s.match == nil {
			if value, ok := m.Get(s.key); ok && matchValueAlways(value) {
				return false
			}
		} else if value, ok := m.Get(s.key); !ok || !s.match(value) {
			return false
		}
	}
	return true
}
