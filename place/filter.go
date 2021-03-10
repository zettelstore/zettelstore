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
	"io"
	"sort"
	"strings"
	"sync"

	"zettelstore.de/z/domain/meta"
)

// Filter specifies a mechanism for selecting zettel.
type Filter struct {
	expr      map[string][]filterValue
	negate    bool
	preFilter func(*meta.Meta) bool
	compiled  filterFunc
	mx        sync.RWMutex
}

type filterValue struct {
	value  string
	negate bool
}

// AddExpr adds a match expression to the filter.
func (f *Filter) AddExpr(key, val string, negate bool) *Filter {
	if f == nil {
		f = new(Filter)
	}
	f.mx.Lock()
	defer f.mx.Unlock()
	if f.expr == nil {
		f.expr = map[string][]filterValue{key: {{value: val, negate: negate}}}
	} else {
		f.expr[key] = append(f.expr[key], filterValue{value: val, negate: negate})
	}
	return f
}

// SetNegate changes the filter to reverse its selection.
func (f *Filter) SetNegate() *Filter {
	if f == nil {
		f = new(Filter)
	}
	f.mx.Lock()
	defer f.mx.Unlock()
	f.negate = true
	return f
}

// AddPreFilter adds the pre-filter selection predicate.
func (f *Filter) AddPreFilter(preFilter func(*meta.Meta) bool) *Filter {
	if f == nil {
		f = new(Filter)
	}
	f.mx.Lock()
	defer f.mx.Unlock()
	if pre := f.preFilter; pre == nil {
		f.preFilter = preFilter
	} else {
		f.preFilter = func(m *meta.Meta) bool {
			return preFilter(m) && pre(m)
		}
	}
	return f
}

// Keys returns the used metadata keys of this filter.
func (f *Filter) Keys() []string {
	if f == nil {
		return nil
	}
	f.mx.RLock()
	defer f.mx.RUnlock()
	var result []string
	for key := range f.expr {
		result = append(result, key)
	}
	return result
}

// Match checks whether the given meta matches the filter specification.
func (f *Filter) Match(m *meta.Meta) bool {
	if f == nil {
		return true
	}
	f.mx.Lock()
	defer f.mx.Unlock()
	if pre := f.preFilter; pre != nil {
		if !pre(m) {
			return false
		}
	}
	if f.compiled == nil {
		f.compiled = compileFilter(f)
	}
	return f.compiled(m) != f.negate
}

type filterFunc func(*meta.Meta) bool

func filterNone(m *meta.Meta) bool { return true }

type matchFunc func(value string) bool

func matchNever(value string) bool  { return false }
func matchAlways(value string) bool { return true }

type matchSpec struct {
	key   string
	match matchFunc
}

// compileFilter calculates a filter func based on the given filter.
func compileFilter(filter *Filter) filterFunc {
	specs, nomatch, searchAll := createFilterSpecs(filter)
	if len(specs) == 0 && len(nomatch) == 0 {
		if searchAll == nil {
			return filterNone
		}
		return searchAll
	}
	searchMeta := makeSearchMetaFilterFunc(specs, nomatch)
	if searchAll == nil {
		return searchMeta
	}
	return func(m *meta.Meta) bool {
		return searchAll(m) || searchMeta(m)
	}
}

func createFilterSpecs(filter *Filter) ([]matchSpec, []string, filterFunc) {
	specs := make([]matchSpec, 0, len(filter.expr))
	var nomatch []string
	var searchAll filterFunc
	for key, values := range filter.expr {
		if key == "" {
			// Special handling if searching all keys...
			searchAll = createSearchAllFunc(values, filter.negate)
			continue
		}
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
	return specs, nomatch, searchAll
}

func hasEmptyValues(values []filterValue) (bool, int) {
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

func createMatchFunc(key string, values []filterValue) matchFunc {
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

func createMatchBoolFunc(values []filterValue) matchFunc {
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

func createMatchIDFunc(values []filterValue) matchFunc {
	return func(value string) bool {
		for _, v := range values {
			if strings.HasPrefix(value, v.value) == v.negate {
				return false
			}
		}
		return true
	}
}

func createMatchIDSetFunc(values []filterValue) matchFunc {
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

func createMatchTagSetFunc(values []filterValue) matchFunc {
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

func createMatchWordFunc(values []filterValue) matchFunc {
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

func createMatchWordSetFunc(values []filterValue) matchFunc {
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

func createMatchStringFunc(values []filterValue) matchFunc {
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

func createSearchAllFunc(values []filterValue, negate bool) filterFunc {
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

func makeSearchMetaFilterFunc(specs []matchSpec, nomatch []string) filterFunc {
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

// createBoolSearchFunc only creates a matchFunc if the values to compare are
// possible bool values. Otherwise every meta with a bool key could match the
// search query.
func createBoolSearchFunc(key string, values []filterValue) matchFunc {
	for _, v := range values {
		if len(v.value) > 0 && !strings.ContainsRune("01tfTFynYN", rune(v.value[0])) {
			return matchNever
		}
	}
	return createMatchFunc(key, values)
}

func sliceToLower(sl []filterValue) []filterValue {
	result := make([]filterValue, 0, len(sl))
	for _, s := range sl {
		result = append(result, filterValue{
			value:  strings.ToLower(s.value),
			negate: s.negate,
		})
	}
	return result
}

func preprocessSet(set []filterValue) [][]filterValue {
	result := make([][]filterValue, 0, len(set))
	for _, elem := range set {
		splitElems := strings.Split(elem.value, ",")
		valueElems := make([]filterValue, 0, len(splitElems))
		for _, se := range splitElems {
			e := strings.TrimSpace(se)
			if len(e) > 0 {
				valueElems = append(valueElems, filterValue{value: e, negate: elem.negate})
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

// Print the filter to a writer.
func (f *Filter) Print(w io.Writer) {
	if f.negate {
		io.WriteString(w, "NOT (")
	}
	names := make([]string, 0, len(f.expr))
	for name := range f.expr {
		names = append(names, name)
	}
	sort.Strings(names)
	for i, name := range names {
		if i > 0 {
			io.WriteString(w, " AND ")
		}
		if name == "" {
			io.WriteString(w, "ANY")
		} else {
			io.WriteString(w, name)
		}
		printFilterExprValues(w, f.expr[name])
	}
	if f.negate {
		io.WriteString(w, ")")
	}
}

func printFilterExprValues(w io.Writer, values []filterValue) {
	if len(values) == 0 {
		io.WriteString(w, " MATCH ANY")
		return
	}

	for j, val := range values {
		if j > 0 {
			io.WriteString(w, " AND")
		}
		if val.negate {
			io.WriteString(w, " NOT")
		}
		io.WriteString(w, " MATCH ")
		if val.value == "" {
			io.WriteString(w, "ANY")
		} else {
			io.WriteString(w, val.value)
		}
	}
}
