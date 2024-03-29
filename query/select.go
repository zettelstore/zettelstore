//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2020-present Detlef Stern
//-----------------------------------------------------------------------------

package query

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"zettelstore.de/z/encoder/textenc"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/strfun"
	"zettelstore.de/z/zettel/meta"
)

type matchValueFunc func(value string) bool

func matchValueNever(string) bool { return false }

type matchSpec struct {
	key   string
	match matchValueFunc
}

// compileMeta calculates a selection func based on the given select criteria.
func (ct *conjTerms) compileMeta() MetaMatchFunc {
	for key, vals := range ct.mvals {
		// All queried keys must exist, if there is at least one non-negated compare operation
		//
		// This is only an optimization to make selection of metadata faster.
		if countNegatedOps(vals) < len(vals) {
			ct.addKey(key, cmpExist)
		}
	}
	for _, op := range ct.keys {
		if op != cmpExist && op != cmpNotExist {
			return matchNever
		}
	}
	posSpecs, negSpecs := ct.createSelectSpecs()
	if len(posSpecs) > 0 || len(negSpecs) > 0 || len(ct.keys) > 0 {
		return makeSearchMetaMatchFunc(posSpecs, negSpecs, ct.keys)
	}
	return nil
}

func countNegatedOps(vals []expValue) int {
	count := 0
	for _, val := range vals {
		if val.op.isNegated() {
			count++
		}
	}
	return count
}

func (ct *conjTerms) createSelectSpecs() (posSpecs, negSpecs []matchSpec) {
	posSpecs = make([]matchSpec, 0, len(ct.mvals))
	negSpecs = make([]matchSpec, 0, len(ct.mvals))
	for key, values := range ct.mvals {
		if !meta.KeyIsValid(key) {
			continue
		}
		posMatch, negMatch := createPosNegMatchFunc(key, values, ct.addSearch)
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

func noAddSearch(expValue) { /* Just does nothing, for negated queries */ }

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
	if meta.IsProperty(key) {
		// Properties are not stored in the Zettelstore and in the search index.
		addSearch = noAddSearch
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
	case meta.TypeID:
		return createMatchIDFunc(values, addSearch)
	case meta.TypeIDSet:
		return createMatchIDSetFunc(values, addSearch)
	case meta.TypeTimestamp:
		return createMatchTimestampFunc(values, addSearch)
	case meta.TypeNumber:
		return createMatchNumberFunc(values, addSearch)
	case meta.TypeTagSet:
		return createMatchTagSetFunc(values, addSearch)
	case meta.TypeWord:
		return createMatchWordFunc(values, addSearch)
	case meta.TypeZettelmarkup:
		return createMatchZmkFunc(values, addSearch)
	}
	return createMatchStringFunc(values, addSearch)
}

func createMatchIDFunc(values []expValue, addSearch addSearchFunc) matchValueFunc {
	preds := valuesToIDPredicates(values, addSearch)
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
	predList := valuesToSetPredicates(preprocessSet(values), addSearch)
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
func createMatchTimestampFunc(values []expValue, addSearch addSearchFunc) matchValueFunc {
	preds := valuesToTimestampPredicates(values, addSearch)
	return func(value string) bool {
		value = meta.ExpandTimestamp(value)
		for _, pred := range preds {
			if !pred(value) {
				return false
			}
		}
		return true
	}
}

func createMatchNumberFunc(values []expValue, addSearch addSearchFunc) matchValueFunc {
	preds := valuesToNumberPredicates(values, addSearch)
	return func(value string) bool {
		for _, pred := range preds {
			if !pred(value) {
				return false
			}
		}
		return true
	}
}

func createMatchTagSetFunc(values []expValue, addSearch addSearchFunc) matchValueFunc {
	predList := valuesToSetPredicates(processTagSet(preprocessSet(sliceToLower(values))), addSearch)
	return func(value string) bool {
		tags := meta.TagsFromValue(value)
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
	preds := valuesToWordPredicates(sliceToLower(values), addSearch)
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

func createMatchStringFunc(values []expValue, addSearch addSearchFunc) matchValueFunc {
	preds := valuesToStringPredicates(sliceToLower(values), addSearch)
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

func createMatchZmkFunc(values []expValue, addSearch addSearchFunc) matchValueFunc {
	normPreds := make([]stringPredicate, 0, len(values))
	negPreds := make([]stringPredicate, 0, len(values))
	for _, v := range values {
		for _, word := range strfun.NormalizeWords(v.value) {
			if cmpOp := v.op; cmpOp.isNegated() {
				cmpOp = cmpOp.negate()
				negPreds = append(negPreds, createWordCompareFunc(word, cmpOp))
			} else {
				normPreds = append(normPreds, createWordCompareFunc(word, cmpOp))
				addSearch(expValue{word, cmpOp}) // addSearch only for positive selections
			}
		}
	}
	return func(metaValue string) bool {
		temp := strings.Fields(zmk2text(metaValue))
		values := make([]string, 0, len(temp))
		for _, s := range temp {
			values = append(values, strfun.NormalizeWords(s)...)
		}
		for _, pred := range normPreds {
			if noneOf(pred, values) {
				return false
			}
		}
		for _, pred := range negPreds {
			for _, val := range values {
				if pred(val) {
					return false
				}
			}
		}
		return true
	}
}

func noneOf(pred stringPredicate, values []string) bool {
	for _, value := range values {
		if pred(value) {
			return false
		}
	}
	return true
}

func zmk2text(zmk string) string {
	isASCII, hasUpper, needParse := true, false, false
	for i := range len(zmk) {
		ch := zmk[i]
		if ch >= utf8.RuneSelf {
			isASCII = false
			break
		}
		hasUpper = hasUpper || ('A' <= ch && ch <= 'Z')
		needParse = needParse || !(('A' <= ch && ch <= 'Z') || ('a' <= ch && ch <= 'z') || ('0' <= ch && ch <= '9') || ch == ' ')
	}
	if isASCII {
		if !needParse {
			if !hasUpper {
				return zmk
			}
			return strings.ToLower(zmk)
		}
	}
	is := parser.ParseMetadata(zmk)
	var sb strings.Builder
	if _, err := textenc.Create().WriteInlines(&sb, &is); err != nil {
		return strings.ToLower(zmk)
	}
	return strings.ToLower(sb.String())
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

func valuesToIDPredicates(values []expValue, addSearch addSearchFunc) []stringPredicate {
	result := make([]stringPredicate, len(values))
	for i, v := range values {
		value := v.value
		if len(value) > 14 {
			value = value[:14]
		}
		switch op := disambiguatedIDOp(v.op); op {
		case cmpLess, cmpNoLess, cmpGreater, cmpNoGreater:
			if isDigits(value) {
				// Never add the strValue to search.
				// Append enough zeroes to make it comparable as string.
				// (an ID and a timestamp always have 14 digits)
				strValue := value + "00000000000000"[:14-len(value)]
				result[i] = createIDCompareFunc(strValue, op)
				continue
			}
			fallthrough
		default:
			// Otherwise compare as a word.
			if !op.isNegated() {
				addSearch(v) // addSearch only for positive selections
			}
			result[i] = createWordCompareFunc(value, op)
		}
	}
	return result
}

func isDigits(s string) bool {
	for i := range len(s) {
		if ch := s[i]; ch < '0' || '9' < ch {
			return false
		}
	}
	return true
}

func disambiguatedIDOp(cmpOp compareOp) compareOp { return disambiguateWordOp(cmpOp) }

func createIDCompareFunc(cmpVal string, cmpOp compareOp) stringPredicate {
	return createWordCompareFunc(cmpVal, cmpOp)
}

func valuesToTimestampPredicates(values []expValue, addSearch addSearchFunc) []stringPredicate {
	result := make([]stringPredicate, len(values))
	for i, v := range values {
		value := meta.ExpandTimestamp(v.value)
		switch op := disambiguatedTimestampOp(v.op); op {
		case cmpLess, cmpNoLess, cmpGreater, cmpNoGreater:
			if isDigits(value) {
				// Never add the value to search.
				result[i] = createTimestampCompareFunc(value, op)
				continue
			}
			fallthrough
		default:
			// Otherwise compare as a word.
			if !op.isNegated() {
				addSearch(v) // addSearch only for positive selections
			}
			result[i] = createWordCompareFunc(value, op)
		}
	}
	return result
}

func disambiguatedTimestampOp(cmpOp compareOp) compareOp { return disambiguateWordOp(cmpOp) }

func createTimestampCompareFunc(cmpVal string, cmpOp compareOp) stringPredicate {
	return createWordCompareFunc(cmpVal, cmpOp)
}

func valuesToNumberPredicates(values []expValue, addSearch addSearchFunc) []stringPredicate {
	result := make([]stringPredicate, len(values))
	for i, v := range values {
		switch op := disambiguatedNumberOp(v.op); op {
		case cmpEqual, cmpNotEqual, cmpLess, cmpNoLess, cmpGreater, cmpNoGreater:
			iValue, err := strconv.ParseInt(v.value, 10, 64)
			if err == nil {
				// Never add the strValue to search.
				result[i] = createNumberCompareFunc(iValue, op)
				continue
			}
			fallthrough
		default:
			// In all other cases, a number is treated like a word.
			if !op.isNegated() {
				addSearch(v) // addSearch only for positive selections
			}
			result[i] = createWordCompareFunc(v.value, op)
		}
	}
	return result
}

func disambiguatedNumberOp(cmpOp compareOp) compareOp { return disambiguateWordOp(cmpOp) }

func createNumberCompareFunc(cmpVal int64, cmpOp compareOp) stringPredicate {
	var cmpFunc func(int64) bool
	switch cmpOp {
	case cmpEqual:
		cmpFunc = func(iMetaVal int64) bool { return iMetaVal == cmpVal }
	case cmpNotEqual:
		cmpFunc = func(iMetaVal int64) bool { return iMetaVal != cmpVal }
	case cmpLess:
		cmpFunc = func(iMetaVal int64) bool { return iMetaVal < cmpVal }
	case cmpNoLess:
		cmpFunc = func(iMetaVal int64) bool { return iMetaVal >= cmpVal }
	case cmpGreater:
		cmpFunc = func(iMetaVal int64) bool { return iMetaVal > cmpVal }
	case cmpNoGreater:
		cmpFunc = func(iMetaVal int64) bool { return iMetaVal <= cmpVal }
	default:
		panic(fmt.Sprintf("Unknown compare operation %d with value %q", cmpOp, cmpVal))
	}
	return func(metaVal string) bool {
		iMetaVal, err := strconv.ParseInt(metaVal, 10, 64)
		if err != nil {
			return false
		}
		return cmpFunc(iMetaVal)
	}
}

func valuesToStringPredicates(values []expValue, addSearch addSearchFunc) []stringPredicate {
	result := make([]stringPredicate, len(values))
	for i, v := range values {
		op := disambiguatedStringOp(v.op)
		if !op.isNegated() {
			addSearch(v) // addSearch only for positive selections
		}
		result[i] = createStringCompareFunc(v.value, op)
	}
	return result
}

func disambiguatedStringOp(cmpOp compareOp) compareOp {
	switch cmpOp {
	case cmpHas:
		return cmpMatch
	case cmpHasNot:
		return cmpNoMatch
	default:
		return cmpOp
	}
}

func createStringCompareFunc(cmpVal string, cmpOp compareOp) stringPredicate {
	return createWordCompareFunc(cmpVal, cmpOp)
}

func valuesToWordPredicates(values []expValue, addSearch addSearchFunc) []stringPredicate {
	result := make([]stringPredicate, len(values))
	for i, v := range values {
		op := disambiguateWordOp(v.op)
		if !op.isNegated() {
			addSearch(v) // addSearch only for positive selections
		}
		result[i] = createWordCompareFunc(v.value, op)
	}
	return result
}

func disambiguateWordOp(cmpOp compareOp) compareOp {
	switch cmpOp {
	case cmpHas:
		return cmpEqual
	case cmpHasNot:
		return cmpNotEqual
	default:
		return cmpOp
	}
}

func createWordCompareFunc(cmpVal string, cmpOp compareOp) stringPredicate {
	switch cmpOp {
	case cmpEqual:
		return func(metaVal string) bool { return metaVal == cmpVal }
	case cmpNotEqual:
		return func(metaVal string) bool { return metaVal != cmpVal }
	case cmpPrefix:
		return func(metaVal string) bool { return strings.HasPrefix(metaVal, cmpVal) }
	case cmpNoPrefix:
		return func(metaVal string) bool { return !strings.HasPrefix(metaVal, cmpVal) }
	case cmpSuffix:
		return func(metaVal string) bool { return strings.HasSuffix(metaVal, cmpVal) }
	case cmpNoSuffix:
		return func(metaVal string) bool { return !strings.HasSuffix(metaVal, cmpVal) }
	case cmpMatch:
		return func(metaVal string) bool { return strings.Contains(metaVal, cmpVal) }
	case cmpNoMatch:
		return func(metaVal string) bool { return !strings.Contains(metaVal, cmpVal) }
	case cmpLess:
		return func(metaVal string) bool { return metaVal < cmpVal }
	case cmpNoLess:
		return func(metaVal string) bool { return metaVal >= cmpVal }
	case cmpGreater:
		return func(metaVal string) bool { return metaVal > cmpVal }
	case cmpNoGreater:
		return func(metaVal string) bool { return metaVal <= cmpVal }
	case cmpHas, cmpHasNot:
		panic(fmt.Sprintf("operator %d not disambiguated with value %q", cmpOp, cmpVal))
	default:
		panic(fmt.Sprintf("Unknown compare operation %d with value %q", cmpOp, cmpVal))
	}
}

type stringSetPredicate func(value []string) bool

func valuesToSetPredicates(values [][]expValue, addSearch addSearchFunc) [][]stringSetPredicate {
	result := make([][]stringSetPredicate, len(values))
	for i, val := range values {
		elemPreds := make([]stringSetPredicate, len(val))
		for j, v := range val {
			opVal := v.value // loop variable is used in closure --> save needed value
			switch op := disambiguateWordOp(v.op); op {
			case cmpEqual:
				addSearch(v) // addSearch only for positive selections
				fallthrough
			case cmpNotEqual:
				elemPreds[j] = makeStringSetPredicate(opVal, stringEqual, op == cmpEqual)
			case cmpPrefix:
				addSearch(v)
				fallthrough
			case cmpNoPrefix:
				elemPreds[j] = makeStringSetPredicate(opVal, strings.HasPrefix, op == cmpPrefix)
			case cmpSuffix:
				addSearch(v)
				fallthrough
			case cmpNoSuffix:
				elemPreds[j] = makeStringSetPredicate(opVal, strings.HasSuffix, op == cmpSuffix)
			case cmpMatch:
				addSearch(v)
				fallthrough
			case cmpNoMatch:
				elemPreds[j] = makeStringSetPredicate(opVal, strings.Contains, op == cmpMatch)
			case cmpLess, cmpNoLess:
				elemPreds[j] = makeStringSetPredicate(opVal, stringLess, op == cmpLess)
			case cmpGreater, cmpNoGreater:
				elemPreds[j] = makeStringSetPredicate(opVal, stringGreater, op == cmpGreater)
			case cmpHas, cmpHasNot:
				panic(fmt.Sprintf("operator %d not disambiguated with value %q", op, opVal))
			default:
				panic(fmt.Sprintf("Unknown compare operation %d with value %q", op, opVal))
			}
		}
		result[i] = elemPreds
	}
	return result
}

func stringEqual(val1, val2 string) bool   { return val1 == val2 }
func stringLess(val1, val2 string) bool    { return val1 < val2 }
func stringGreater(val1, val2 string) bool { return val1 > val2 }

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
