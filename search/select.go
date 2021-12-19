//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
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
func compileMeta(tags expTagValues) MetaMatchFunc {
	posSpecs, negSpecs, nomatch := createSelectSpecs(tags)
	if len(posSpecs) > 0 || len(negSpecs) > 0 || len(nomatch) > 0 {
		return makeSearchMetaMatchFunc(posSpecs, negSpecs, nomatch)
	}
	return nil
}

func createSelectSpecs(tags map[string][]expValue) (posSpecs, negSpecs []matchSpec, nomatch []string) {
	posSpecs = make([]matchSpec, 0, len(tags))
	negSpecs = make([]matchSpec, 0, len(tags))
	for key, values := range tags {
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
		posMatch, negMatch := createPosNegMatchFunc(key, values)
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
			if v.negate {
				never++
			} else {
				always++
			}
		}
	}
	return always, never
}

func createPosNegMatchFunc(key string, values []expValue) (posMatch, negMatch matchValueFunc) {
	posValues := make([]opValue, 0, len(values))
	negValues := make([]opValue, 0, len(values))
	for _, val := range values {
		if val.negate {
			negValues = append(negValues, opValue{value: val.value, op: val.op.negate()})
		} else {
			posValues = append(posValues, opValue{value: val.value, op: val.op})
		}
	}
	return createMatchFunc(key, posValues), createMatchFunc(key, negValues)
}

// opValue is an expValue, but w/o the field "negate"
type opValue struct {
	value string
	op    compareOp
}

func createMatchFunc(key string, values []opValue) matchValueFunc {
	if len(values) == 0 {
		return nil
	}
	switch meta.Type(key) {
	case meta.TypeBool:
		return createMatchBoolFunc(values)
	case meta.TypeCredential:
		return matchValueNever
	case meta.TypeID, meta.TypeTimestamp: // ID and timestamp use the same layout
		return createMatchIDFunc(values)
	case meta.TypeIDSet:
		return createMatchIDSetFunc(values)
	case meta.TypeTagSet:
		return createMatchTagSetFunc(values)
	case meta.TypeWord:
		return createMatchWordFunc(values)
	case meta.TypeWordSet:
		return createMatchWordSetFunc(values)
	}
	return createMatchStringFunc(values)
}

type boolPredicate func(bool) bool

func boolSame(value bool) bool   { return value }
func boolNegate(value bool) bool { return value }

func createMatchBoolFunc(values []opValue) matchValueFunc {
	preds := make([]boolPredicate, len(values))
	for i, v := range values {
		positiveTest := false
		switch v.op {
		case cmpDefault, cmpEqual, cmpPrefix, cmpSuffix, cmpContains:
			positiveTest = true
		case cmpNotDefault, cmpNotEqual, cmpNoPrefix, cmpNoSuffix, cmpNotContains:
			// positiveTest = false
		default:
			panic(fmt.Sprintf("Unknown compare operation %d", v.op))
		}
		bValue := meta.BoolValue(v.value)
		if positiveTest == bValue {
			preds[i] = boolSame
		} else {
			preds[i] = boolNegate
		}
	}
	return func(value string) bool {
		bValue := meta.BoolValue(value)
		for _, pred := range preds {
			if !pred(bValue) {
				return false
			}
		}
		return true
	}
}

func createMatchIDFunc(values []opValue) matchValueFunc {
	preds := valuesToStringPredicates(values, cmpPrefix)
	return func(value string) bool {
		for _, pred := range preds {
			if !pred(value) {
				return false
			}
		}
		return true
	}
}

func createMatchIDSetFunc(values []opValue) matchValueFunc {
	predList := valuesToStringSetPredicates(preprocessSet(values), cmpPrefix)
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

func createMatchTagSetFunc(values []opValue) matchValueFunc {
	predList := valuesToStringSetPredicates(processTagSet(preprocessSet(sliceToLower(values))), cmpEqual)
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

func processTagSet(valueSet [][]opValue) [][]opValue {
	result := make([][]opValue, len(valueSet))
	for i, values := range valueSet {
		tags := make([]opValue, len(values))
		for j, val := range values {
			if tval := val.value; tval != "" && tval[0] == '#' {
				tval = meta.CleanTag(tval)
				tags[j] = opValue{value: tval, op: resolveDefaultOp(val.op, cmpEqual)}
			} else {
				tags[j] = opValue{value: tval, op: resolveDefaultOp(val.op, cmpPrefix)}
			}
		}
		result[i] = tags
	}
	return result
}

func createMatchWordFunc(values []opValue) matchValueFunc {
	preds := valuesToStringPredicates(sliceToLower(values), cmpEqual)
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

func createMatchWordSetFunc(values []opValue) matchValueFunc {
	predsList := valuesToStringSetPredicates(preprocessSet(sliceToLower(values)), cmpEqual)
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

func createMatchStringFunc(values []opValue) matchValueFunc {
	preds := valuesToStringPredicates(sliceToLower(values), cmpContains)
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

func sliceToLower(sl []opValue) []opValue {
	result := make([]opValue, 0, len(sl))
	for _, s := range sl {
		result = append(result, opValue{
			value: strings.ToLower(s.value),
			op:    s.op,
		})
	}
	return result
}

func preprocessSet(set []opValue) [][]opValue {
	result := make([][]opValue, 0, len(set))
	for _, elem := range set {
		splitElems := strings.Split(elem.value, ",")
		valueElems := make([]opValue, 0, len(splitElems))
		for _, se := range splitElems {
			e := strings.TrimSpace(se)
			if len(e) > 0 {
				valueElems = append(valueElems, opValue{value: e, op: elem.op})
			}
		}
		if len(valueElems) > 0 {
			result = append(result, valueElems)
		}
	}
	return result
}

type stringPredicate func(string) bool

func valuesToStringPredicates(values []opValue, defOp compareOp) []stringPredicate {
	result := make([]stringPredicate, len(values))
	for i, v := range values {
		opVal := v.value // loop variable is used in closure --> save needed value
		switch op := resolveDefaultOp(v.op, defOp); op {
		case cmpEqual:
			result[i] = func(metaVal string) bool { return metaVal == opVal }
		case cmpNotEqual:
			result[i] = func(metaVal string) bool { return metaVal != opVal }
		case cmpPrefix:
			result[i] = func(metaVal string) bool { return strings.HasPrefix(metaVal, opVal) }
		case cmpNoPrefix:
			result[i] = func(metaVal string) bool { return !strings.HasPrefix(metaVal, opVal) }
		case cmpSuffix:
			result[i] = func(metaVal string) bool { return strings.HasSuffix(metaVal, opVal) }
		case cmpNoSuffix:
			result[i] = func(metaVal string) bool { return !strings.HasSuffix(metaVal, opVal) }
		case cmpContains:
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

func valuesToStringSetPredicates(values [][]opValue, defOp compareOp) [][]stringSetPredicate {
	result := make([][]stringSetPredicate, len(values))
	for i, val := range values {
		elemPreds := make([]stringSetPredicate, len(val))
		for j, v := range val {
			opVal := v.value // loop variable is used in closure --> save needed value
			switch op := resolveDefaultOp(v.op, defOp); op {
			case cmpEqual:
				elemPreds[j] = makeStringSetPredicate(opVal, stringEqual, true)
			case cmpNotEqual:
				elemPreds[j] = makeStringSetPredicate(opVal, stringEqual, false)
			case cmpPrefix:
				elemPreds[j] = makeStringSetPredicate(opVal, strings.HasPrefix, true)
			case cmpNoPrefix:
				elemPreds[j] = makeStringSetPredicate(opVal, strings.HasPrefix, false)
			case cmpSuffix:
				elemPreds[j] = makeStringSetPredicate(opVal, strings.HasSuffix, true)
			case cmpNoSuffix:
				elemPreds[j] = makeStringSetPredicate(opVal, strings.HasSuffix, false)
			case cmpContains:
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

func resolveDefaultOp(op, defOp compareOp) compareOp {
	if op == cmpDefault {
		return defOp
	}
	if op == cmpNotDefault {
		return defOp.negate()
	}
	return op
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
			if _, ok := m.Get(s.key); ok {
				return false
			}
		} else if value, ok := m.Get(s.key); !ok || !s.match(value) {
			return false
		}
	}
	return true
}
