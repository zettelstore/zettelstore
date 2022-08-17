//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package search provides a zettel search.
package search

import (
	"strconv"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/meta"
)

type sortFunc func(i, j int) bool

func createSortFunc(order []sortOrder, ml []*meta.Meta) sortFunc {
	hasID := false
	sortFuncs := make([]sortFunc, 0, len(order)+1)
	for _, o := range order {
		sortFuncs = append(sortFuncs, createOneSortFunc(o.key, o.descending, ml))
		if o.key == api.KeyID {
			hasID = true
			break
		}
	}
	if !hasID {
		sortFuncs = append(sortFuncs, func(i, j int) bool { return ml[i].Zid > ml[j].Zid })
	}
	// return sortFuncs[0]
	if len(sortFuncs) == 1 {
		return sortFuncs[0]
	}
	return func(i, j int) bool {
		for _, sf := range sortFuncs {
			if sf(i, j) {
				return true
			}
			if sf(j, i) {
				return false
			}
		}
		return false
	}
}

func createOneSortFunc(key string, descending bool, ml []*meta.Meta) sortFunc {
	keyType := meta.Type(key)
	if key == api.KeyID || keyType == meta.TypeCredential {
		if descending {
			return func(i, j int) bool { return ml[i].Zid > ml[j].Zid }
		}
		return func(i, j int) bool { return ml[i].Zid < ml[j].Zid }
	}
	if keyType == meta.TypeNumber {
		return createSortNumberFunc(ml, key, descending)
	}
	return createSortStringFunc(ml, key, descending)
}

func createSortNumberFunc(ml []*meta.Meta, key string, descending bool) sortFunc {
	if descending {
		return func(i, j int) bool {
			iVal, iOk := getNum(ml[i], key)
			jVal, jOk := getNum(ml[j], key)
			return (iOk && (!jOk || iVal > jVal)) || !jOk
		}
	}
	return func(i, j int) bool {
		iVal, iOk := getNum(ml[i], key)
		jVal, jOk := getNum(ml[j], key)
		return (iOk && (!jOk || iVal < jVal)) || !jOk
	}
}

func createSortStringFunc(ml []*meta.Meta, key string, descending bool) sortFunc {
	if descending {
		return func(i, j int) bool {
			iVal, iOk := ml[i].Get(key)
			jVal, jOk := ml[j].Get(key)
			return (iOk && (!jOk || iVal > jVal)) || !jOk
		}
	}
	return func(i, j int) bool {
		iVal, iOk := ml[i].Get(key)
		jVal, jOk := ml[j].Get(key)
		return (iOk && (!jOk || iVal < jVal)) || !jOk
	}
}

func getNum(m *meta.Meta, key string) (int64, bool) {
	if s, ok := m.Get(key); ok {
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i, true
		}
	}
	return 0, false
}
