//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package search provides a zettel search.
package search

import (
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/meta"
)

type matchFunc func(value string) bool

func matchNever(value string) bool  { return false }
func matchAlways(value string) bool { return true }

type matchSpec struct {
	key   string
	match matchFunc
}

// compileSelect calculates a selection func based on the given select criteria.
func compileSelect(tags expTagValues) MetaMatchFunc {
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
				posSpecs = append(posSpecs, matchSpec{key, matchAlways})
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
		if v.value != "" {
			continue
		}
		if v.negate {
			never++
		} else {
			always++
		}
	}
	return always, never
}

func createPosNegMatchFunc(key string, values []expValue) (posMatch, negMatch matchFunc) {
	posValues := make([]opValue, 0, len(values))
	negValues := make([]opValue, 0, len(values))
	for _, val := range values {
		if val.negate {
			negValues = append(negValues, opValue{value: val.value, op: val.op})
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

func createMatchFunc(key string, values []opValue) matchFunc {
	if len(values) == 0 {
		return nil
	}
	switch meta.Type(key) {
	case meta.TypeBool:
		return createMatchBoolFunc(values)
	case meta.TypeCredential:
		return matchNever
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

func createMatchBoolFunc(values []opValue) matchFunc {
	preValues := make([]bool, 0, len(values))
	for _, v := range values {
		preValues = append(preValues, meta.BoolValue(v.value))
	}
	return func(value string) bool {
		bValue := meta.BoolValue(value)
		for _, v := range preValues {
			if bValue != v {
				return false
			}
		}
		return true
	}
}

func createMatchIDFunc(values []opValue) matchFunc {
	return func(value string) bool {
		for _, v := range values {
			if !strings.HasPrefix(value, v.value) {
				return false
			}
		}
		return true
	}
}

func createMatchIDSetFunc(values []opValue) matchFunc {
	idValues := preprocessSet(sliceToLower(values))
	return func(value string) bool {
		ids := meta.ListFromValue(value)
		for _, neededIDs := range idValues {
			for _, neededID := range neededIDs {
				if !matchAllID(ids, neededID.value) {
					return false
				}
			}
		}
		return true
	}
}

func matchAllID(zettelIDs []string, neededID string) bool {
	for _, zt := range zettelIDs {
		if strings.HasPrefix(zt, neededID) {
			return true
		}
	}
	return false
}

func createMatchTagSetFunc(values []opValue) matchFunc {
	tagValues := processTagSet(preprocessSet(sliceToLower(values)))
	return func(value string) bool {
		tags := meta.ListFromValue(value)
		// Remove leading '#' from each tag
		for i, tag := range tags {
			tags[i] = meta.CleanTag(tag)
		}
		for _, neededTags := range tagValues {
			for _, neededTag := range neededTags {
				if !matchAllTag(tags, neededTag.value, neededTag.equal) {
					return false
				}
			}
		}
		return true
	}
}

type tagQueryValue struct {
	value string
	equal bool // not equal == prefix
}

func processTagSet(valueSet [][]opValue) [][]tagQueryValue {
	result := make([][]tagQueryValue, len(valueSet))
	for i, values := range valueSet {
		tags := make([]tagQueryValue, len(values))
		for j, val := range values {
			if tval := val.value; tval != "" && tval[0] == '#' {
				tval = meta.CleanTag(tval)
				tags[j] = tagQueryValue{value: tval, equal: true}
			} else {
				tags[j] = tagQueryValue{value: tval, equal: false}
			}
		}
		result[i] = tags
	}
	return result
}

func matchAllTag(zettelTags []string, neededTag string, equal bool) bool {
	if equal {
		for _, zt := range zettelTags {
			if zt == neededTag {
				return true
			}
		}
	} else {
		for _, zt := range zettelTags {
			if strings.HasPrefix(zt, neededTag) {
				return true
			}
		}
	}
	return false
}

func createMatchWordFunc(values []opValue) matchFunc {
	values = sliceToLower(values)
	return func(value string) bool {
		value = strings.ToLower(value)
		for _, v := range values {
			if value != v.value {
				return false
			}
		}
		return true
	}
}

func createMatchWordSetFunc(values []opValue) matchFunc {
	wordValues := preprocessSet(sliceToLower(values))
	return func(value string) bool {
		words := meta.ListFromValue(value)
		for _, neededWords := range wordValues {
			for _, neededWord := range neededWords {
				if !matchAllWord(words, neededWord.value) {
					return false
				}
			}
		}
		return true
	}
}

func createMatchStringFunc(values []opValue) matchFunc {
	values = sliceToLower(values)
	return func(value string) bool {
		value = strings.ToLower(value)
		for _, v := range values {
			if !strings.Contains(value, v.value) {
				return false
			}
		}
		return true
	}
}

func makeSearchMetaMatchFunc(posSpecs, negSpecs []matchSpec, nomatch []string) MetaMatchFunc {
	return func(m *meta.Meta) bool {
		for _, key := range nomatch {
			if _, ok := getMeta(m, key); ok {
				return false
			}
		}
		for _, s := range posSpecs {
			if value, ok := getMeta(m, s.key); !ok || !s.match(value) {
				return false
			}
		}
		for _, s := range negSpecs {
			if s.match == nil {
				if _, ok := m.Get(s.key); ok {
					return false
				}
			} else if value, ok := getMeta(m, s.key); !ok || s.match(value) {
				return false
			}
		}
		return true
	}
}

func getMeta(m *meta.Meta, key string) (string, bool) {
	if key == api.KeyTags {
		return m.Get(api.KeyAllTags)
	}
	return m.Get(key)
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

func matchAllWord(zettelWords []string, neededWord string) bool {
	for _, zw := range zettelWords {
		if zw == neededWord {
			return true
		}
	}
	return false
}
