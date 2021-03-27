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

	"zettelstore.de/z/domain/meta"
)

type matchFunc func(value string) bool

func matchNever(value string) bool  { return false }
func matchAlways(value string) bool { return true }

type matchSpec struct {
	key   string
	match matchFunc
}

// compileFilter calculates a filter func based on the given filter.
func compileFilter(tags expTagValues) MetaMatchFunc {
	specs, nomatch := createFilterSpecs(tags)
	if len(specs) == 0 && len(nomatch) == 0 {
		return nil
	}
	return makeSearchMetaFilterFunc(specs, nomatch)
}

func createFilterSpecs(tags map[string][]expValue) ([]matchSpec, []string) {
	specs := make([]matchSpec, 0, len(tags))
	var nomatch []string
	for key, values := range tags {
		if !meta.KeyIsValid(key) {
			continue
		}
		if empty, negates := hasEmptyValues(values); empty {
			if negates == 0 {
				specs = append(specs, matchSpec{key, matchAlways})
				continue
			}
			if len(values) < negates {
				specs = append(specs, matchSpec{key, matchNever})
				continue
			}
			nomatch = append(nomatch, key)
			continue
		}
		match := createMatchFunc(key, values)
		if match != nil {
			specs = append(specs, matchSpec{key, match})
		}
	}
	return specs, nomatch
}

func hasEmptyValues(values []expValue) (bool, int) {
	var negates int
	for _, v := range values {
		if v.value != "" {
			continue
		}
		if !v.negate {
			return true, 0
		}
		negates++
	}
	return negates > 0, negates
}

func createMatchFunc(key string, values []expValue) matchFunc {
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

func createMatchBoolFunc(values []expValue) matchFunc {
	preValues := make([]bool, 0, len(values))
	for _, v := range values {
		boolValue := meta.BoolValue(v.value)
		if v.negate {
			boolValue = !boolValue
		}
		preValues = append(preValues, boolValue)
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

func createMatchIDFunc(values []expValue) matchFunc {
	return func(value string) bool {
		for _, v := range values {
			if strings.HasPrefix(value, v.value) == v.negate {
				return false
			}
		}
		return true
	}
}

func createMatchIDSetFunc(values []expValue) matchFunc {
	idValues := preprocessSet(sliceToLower(values))
	return func(value string) bool {
		ids := meta.ListFromValue(value)
		for _, neededIDs := range idValues {
			for _, neededID := range neededIDs {
				if matchAllID(ids, neededID.value) == neededID.negate {
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

func createMatchTagSetFunc(values []expValue) matchFunc {
	tagValues := preprocessSet(values)
	return func(value string) bool {
		tags := meta.ListFromValue(value)
		// Remove leading '#' from each tag
		for i, tag := range tags {
			tags[i] = meta.CleanTag(tag)
		}
		for _, neededTags := range tagValues {
			for _, neededTag := range neededTags {
				if matchAllTag(tags, neededTag.value) == neededTag.negate {
					return false
				}
			}
		}
		return true
	}
}

func createMatchWordFunc(values []expValue) matchFunc {
	values = sliceToLower(values)
	return func(value string) bool {
		value = strings.ToLower(value)
		for _, v := range values {
			if (value == v.value) == v.negate {
				return false
			}
		}
		return true
	}
}

func createMatchWordSetFunc(values []expValue) matchFunc {
	wordValues := preprocessSet(sliceToLower(values))
	return func(value string) bool {
		words := meta.ListFromValue(value)
		for _, neededWords := range wordValues {
			for _, neededWord := range neededWords {
				if matchAllWord(words, neededWord.value) == neededWord.negate {
					return false
				}
			}
		}
		return true
	}
}

func createMatchStringFunc(values []expValue) matchFunc {
	values = sliceToLower(values)
	return func(value string) bool {
		value = strings.ToLower(value)
		for _, v := range values {
			if strings.Contains(value, v.value) == v.negate {
				return false
			}
		}
		return true
	}
}

func makeSearchMetaFilterFunc(specs []matchSpec, nomatch []string) MetaMatchFunc {
	return func(m *meta.Meta) bool {
		for _, s := range specs {
			if value, ok := m.Get(s.key); !ok || !s.match(value) {
				return false
			}
		}
		for _, key := range nomatch {
			if _, ok := m.Get(key); ok {
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
			value:  strings.ToLower(s.value),
			negate: s.negate,
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
				valueElems = append(valueElems, expValue{value: e, negate: elem.negate})
			}
		}
		if len(valueElems) > 0 {
			result = append(result, valueElems)
		}
	}
	return result
}

func matchAllTag(zettelTags []string, neededTag string) bool {
	for _, zt := range zettelTags {
		if zt == neededTag {
			return true
		}
	}
	return false
}

func matchAllWord(zettelWords []string, neededWord string) bool {
	for _, zw := range zettelWords {
		if zw == neededWord {
			return true
		}
	}
	return false
}
