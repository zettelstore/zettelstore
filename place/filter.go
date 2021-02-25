//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package place provides a generic interface to zettel places.
package place

import (
	"strings"

	"zettelstore.de/z/domain/meta"
)

// EnsureFilter make sure that there is a current filter.
func EnsureFilter(filter *Filter) *Filter {
	if filter == nil {
		filter = new(Filter)
		filter.Expr = make(FilterExpr)
	}
	return filter
}

// FilterFunc is a predicate to check if given meta must be selected.
type FilterFunc func(*meta.Meta) bool

func selectAll(m *meta.Meta) bool { return true }

type matchFunc func(value string) bool

func matchNever(value string) bool { return false }

type matchSpec struct {
	key   string
	match matchFunc
}

// CreateFilterFunc calculates a filter func based on the given filter.
func CreateFilterFunc(filter *Filter) FilterFunc {
	if filter == nil {
		return selectAll
	}
	specs, searchAll := createFilterSpecs(filter)
	if len(specs) == 0 {
		if searchAll == nil {
			if sel := filter.Select; sel != nil {
				return sel
			}
			return selectAll
		}
		return addSelectFunc(filter, searchAll)
	}
	negate := filter.Negate
	searchMeta := func(m *meta.Meta) bool {
		for _, s := range specs {
			value, ok := m.Get(s.key)
			if !ok || !s.match(value) {
				return negate
			}
		}
		return !negate
	}
	if searchAll == nil {
		return addSelectFunc(filter, searchMeta)
	}
	return addSelectFunc(filter, func(meta *meta.Meta) bool {
		return searchAll(meta) || searchMeta(meta)
	})
}

func createFilterSpecs(filter *Filter) ([]matchSpec, FilterFunc) {
	specs := make([]matchSpec, 0, len(filter.Expr))
	var searchAll FilterFunc
	for key, values := range filter.Expr {
		if key == "" {
			// Special handling if searching all keys...
			searchAll = createSearchAllFunc(values, filter.Negate)
			continue
		}
		if meta.KeyIsValid(key) {
			match := createMatchFunc(key, values)
			if match != nil {
				specs = append(specs, matchSpec{key, match})
			}
		}
	}
	return specs, searchAll
}

func addSelectFunc(filter *Filter, f FilterFunc) FilterFunc {
	if filter == nil {
		return f
	}
	if sel := filter.Select; sel != nil {
		return func(meta *meta.Meta) bool {
			return sel(meta) && f(meta)
		}
	}
	return f
}

func createMatchFunc(key string, values []string) matchFunc {
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

func createMatchBoolFunc(values []string) matchFunc {
	preValues := make([]bool, 0, len(values))
	for _, v := range values {
		preValues = append(preValues, meta.BoolValue(v))
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

func createMatchIDFunc(values []string) matchFunc {
	return func(value string) bool {
		for _, v := range values {
			if !strings.HasPrefix(value, v) {
				return false
			}
		}
		return true
	}
}

func createMatchIDSetFunc(values []string) matchFunc {
	idValues := preprocessSet(sliceToLower(values))
	return func(value string) bool {
		ids := meta.ListFromValue(value)
		for _, neededIDs := range idValues {
			for _, neededID := range neededIDs {
				if !matchAllID(ids, neededID) {
					return false
				}
			}
		}
		return true
	}
}

func createMatchTagSetFunc(values []string) matchFunc {
	tagValues := preprocessSet(values)
	return func(value string) bool {
		tags := meta.ListFromValue(value)
		// Remove leading '#' from each tag
		for i, tag := range tags {
			tags[i] = meta.CleanTag(tag)
		}
		for _, neededTags := range tagValues {
			for _, neededTag := range neededTags {
				if !matchAllTag(tags, neededTag) {
					return false
				}
			}
		}
		return true
	}
}

func createMatchWordFunc(values []string) matchFunc {
	values = sliceToLower(values)
	return func(value string) bool {
		value = strings.ToLower(value)
		for _, v := range values {
			if value != v {
				return false
			}
		}
		return true
	}
}

func createMatchWordSetFunc(values []string) matchFunc {
	wordValues := preprocessSet(sliceToLower(values))
	return func(value string) bool {
		words := meta.ListFromValue(value)
		for _, neededWords := range wordValues {
			for _, neededWord := range neededWords {
				if !matchAllWord(words, neededWord) {
					return false
				}
			}
		}
		return true
	}
}

func createMatchStringFunc(values []string) matchFunc {
	values = sliceToLower(values)
	return func(value string) bool {
		value = strings.ToLower(value)
		for _, v := range values {
			if !strings.Contains(value, v) {
				return false
			}
		}
		return true
	}
}

func createSearchAllFunc(values []string, negate bool) FilterFunc {
	matchFuncs := map[*meta.DescriptionType]matchFunc{}
	return func(m *meta.Meta) bool {
		for _, p := range m.Pairs(true) {
			keyType := meta.Type(p.Key)
			match, ok := matchFuncs[keyType]
			if !ok {
				if keyType == meta.TypeBool {
					match = createBoolSearchFunc(p.Key, values)
				} else {
					match = createMatchFunc(p.Key, values)
				}
				matchFuncs[keyType] = match
			}
			if match(p.Value) {
				return !negate
			}
		}
		match, ok := matchFuncs[meta.Type(meta.KeyID)]
		if !ok {
			match = createMatchFunc(meta.KeyID, values)
		}
		return match(m.Zid.String()) != negate
	}
}

// createBoolSearchFunc only creates a matchFunc if the values to compare are
// possible bool values. Otherwise every meta with a bool key could match the
// search query.
func createBoolSearchFunc(key string, values []string) matchFunc {
	for _, v := range values {
		if len(v) > 0 && !strings.ContainsRune("01tfTFynYN", rune(v[0])) {
			return func(value string) bool { return false }
		}
	}
	return createMatchFunc(key, values)
}

func sliceToLower(sl []string) []string {
	result := make([]string, 0, len(sl))
	for _, s := range sl {
		result = append(result, strings.ToLower(s))
	}
	return result
}

func preprocessSet(set []string) [][]string {
	result := make([][]string, 0, len(set))
	for _, elem := range set {
		splitElems := strings.Split(elem, ",")
		valueElems := make([]string, 0, len(splitElems))
		for _, se := range splitElems {
			e := strings.TrimSpace(se)
			if len(e) > 0 {
				valueElems = append(valueElems, e)
			}
		}
		if len(valueElems) > 0 {
			result = append(result, valueElems)
		}
	}
	return result
}

func matchAllID(zettelIDs []string, neededID string) bool {
	for _, zt := range zettelIDs {
		if strings.HasPrefix(zt, neededID) {
			return true
		}
	}
	return false
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
