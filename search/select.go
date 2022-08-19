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

func matchValueNever(string) bool { return false }

type matchSpec struct {
	key   string
	match matchValueFunc
}

// compileMeta calculates a selection func based on the given select criteria.
func (s *Search) compileMeta() MetaMatchFunc {
	for key := range s.mvals {
		// All queried keys must exist
		s.addKeyExist(key, cmpExist)
	}
	for _, op := range s.keyExist {
		if op != cmpExist && op != cmpNotExist {
			return matchNever
		}
	}
	posSpecs, negSpecs := s.createSelectSpecs()
	if len(posSpecs) > 0 || len(negSpecs) > 0 || len(s.keyExist) > 0 {
		return makeSearchMetaMatchFunc(posSpecs, negSpecs, s.keyExist)
	}
	return nil
}

func (s *Search) createSelectSpecs() (posSpecs, negSpecs []matchSpec) {
	posSpecs = make([]matchSpec, 0, len(s.mvals))
	negSpecs = make([]matchSpec, 0, len(s.mvals))
	for key, values := range s.mvals {
		if !meta.KeyIsValid(key) {
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
	return posSpecs, negSpecs
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
	predList := valuesToStringSetPredicates(processTagSet(preprocessSet(sliceToLower(values))), cmpHas, addSearch)
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
	preds := valuesToStringPredicates(sliceToLower(values), cmpHas, addSearch)
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
	predsList := valuesToStringSetPredicates(preprocessSet(sliceToLower(values)), cmpHas, addSearch)
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
	preds := valuesToStringPredicates(sliceToLower(values), cmpMatch, addSearch)
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
		switch v.op {
		case cmpHas:
			addSearch(v) // addSearch only for positive selections
			result[i] = func(metaVal string) bool { return metaVal == opVal }
		case cmpHasNot:
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
		case cmpMatch:
			addSearch(v)
			result[i] = func(metaVal string) bool { return strings.Contains(metaVal, opVal) }
		case cmpNoMatch:
			result[i] = func(metaVal string) bool { return !strings.Contains(metaVal, opVal) }
		default:
			panic(fmt.Sprintf("Unknown compare operation %d with value %q", v.op, opVal))
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
			switch v.op {
			case cmpHas:
				addSearch(v) // addSearch only for positive selections
				elemPreds[j] = makeStringSetPredicate(opVal, stringEqual, true)
			case cmpHasNot:
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
			case cmpMatch:
				addSearch(v)
				elemPreds[j] = makeStringSetPredicate(opVal, strings.Contains, true)
			case cmpNoMatch:
				elemPreds[j] = makeStringSetPredicate(opVal, strings.Contains, false)
			default:
				panic(fmt.Sprintf("Unknown compare operation %d with value %q", v.op, opVal))
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

func makeSearchMetaMatchFunc(posSpecs, negSpecs []matchSpec, kem keyExistMap) MetaMatchFunc {
	// Optimize: no specs --> just check kwhether key exists
	if len(posSpecs) == 0 && len(negSpecs) == 0 {
		if len(kem) == 0 {
			return nil
		}
		return func(m *meta.Meta) bool { return matchMetaKeyExists(m, kem) }
	}

	// Optimize: only negative or only positive matching
	if len(posSpecs) == 0 {
		return func(m *meta.Meta) bool {
			return matchMetaKeyExists(m, kem) && matchMetaSpecs(m, negSpecs)
		}
	}
	if len(negSpecs) == 0 {
		return func(m *meta.Meta) bool {
			return matchMetaKeyExists(m, kem) && matchMetaSpecs(m, posSpecs)
		}
	}

	return func(m *meta.Meta) bool {
		return matchMetaKeyExists(m, kem) &&
			matchMetaSpecs(m, posSpecs) &&
			matchMetaSpecs(m, negSpecs)
	}
}

func matchMetaKeyExists(m *meta.Meta, kem keyExistMap) bool {
	for key, op := range kem {
		_, found := m.Get(key)
		if found != (op == cmpExist) {
			return false
		}
	}
	return true
}
func matchMetaSpecs(m *meta.Meta, specs []matchSpec) bool {
	for _, s := range specs {
		if value := m.GetDefault(s.key, ""); !s.match(value) {
			return false
		}
	}
	return true
}
