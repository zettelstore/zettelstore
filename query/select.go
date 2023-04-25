//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package query

import (
	"fmt"
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
	for key := range ct.mvals {
		// All queried keys must exist
		ct.addKey(key, cmpExist)
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
	case meta.TypeZettelmarkup:
		return createMatchZmkFunc(values, addSearch)
	}
	return createMatchStringFunc(values, addSearch)
}

func createMatchIDFunc(values []expValue, addSearch addSearchFunc) matchValueFunc {
	preds := valuesToWordPredicates(values, addSearch)
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
	predList := valuesToWordSetPredicates(preprocessSet(values), addSearch)
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
	predList := valuesToWordSetPredicates(processTagSet(preprocessSet(sliceToLower(values))), addSearch)
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

func createMatchWordSetFunc(values []expValue, addSearch addSearchFunc) matchValueFunc {
	predsList := valuesToWordSetPredicates(preprocessSet(sliceToLower(values)), addSearch)
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
	for i := 0; i < len(zmk); i++ {
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
	case cmpHas, cmpHasNot:
		panic(fmt.Sprintf("operator %d not disambiguated with value %q", cmpOp, cmpVal))
	default:
		panic(fmt.Sprintf("Unknown compare operation %d with value %q", cmpOp, cmpVal))
	}
}

type stringSetPredicate func(value []string) bool

func valuesToWordSetPredicates(values [][]expValue, addSearch addSearchFunc) [][]stringSetPredicate {
	result := make([][]stringSetPredicate, len(values))
	for i, val := range values {
		elemPreds := make([]stringSetPredicate, len(val))
		for j, v := range val {
			opVal := v.value // loop variable is used in closure --> save needed value
			switch op := disambiguateWordOp(v.op); op {
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
			case cmpMatch:
				addSearch(v)
				elemPreds[j] = makeStringSetPredicate(opVal, strings.Contains, true)
			case cmpNoMatch:
				elemPreds[j] = makeStringSetPredicate(opVal, strings.Contains, false)
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
