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

func createSortFunc(key string, descending bool, ml []*meta.Meta) sortFunc {
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
