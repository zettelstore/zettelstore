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
	"strconv"

	"zettelstore.de/z/domain/meta"
)

type sortFunc func(i, j int) bool

func createSortFunc(key string, descending bool, ml []*meta.Meta) sortFunc {
	keyType := meta.Type(key)
	if key == meta.KeyID || keyType == meta.TypeCredential {
		if descending {
			return func(i, j int) bool { return ml[i].Zid > ml[j].Zid }
		}
		return func(i, j int) bool { return ml[i].Zid < ml[j].Zid }
	}
	if keyType == meta.TypeBool {
		return createSortBoolFunc(ml, key, descending)
	}
	if keyType == meta.TypeNumber {
		return createSortNumberFunc(ml, key, descending)
	}
	return createSortStringFunc(ml, key, descending)
}

func createSortBoolFunc(ml []*meta.Meta, key string, descending bool) sortFunc {
	if descending {
		return func(i, j int) bool {
			left := ml[i].GetBool(key)
			if left == ml[j].GetBool(key) {
				return i > j
			}
			return left
		}
	}
	return func(i, j int) bool {
		right := ml[j].GetBool(key)
		if ml[i].GetBool(key) == right {
			return i < j
		}
		return right
	}
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

func getNum(m *meta.Meta, key string) (int, bool) {
	if s, ok := m.Get(key); ok {
		if i, err := strconv.Atoi(s); err == nil {
			return i, true
		}
	}
	return 0, false
}
