//-----------------------------------------------------------------------------
// Copyright (c) 2022-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2022-present Detlef Stern
//-----------------------------------------------------------------------------

package meta

import (
	"slices"
	"strings"
)

// Arrangement stores metadata within its categories.
// Typecally a category might be a tag name, a role name, a syntax value.
type Arrangement map[string][]*Meta

// CreateArrangement by inspecting a given key and use the found
// value as a category.
func CreateArrangement(metaList []*Meta, key string) Arrangement {
	if len(metaList) == 0 {
		return nil
	}
	descr := Type(key)
	if descr == nil {
		return nil
	}
	if descr.IsSet {
		return createSetArrangement(metaList, key)
	}
	return createSimplearrangement(metaList, key)
}

func createSetArrangement(metaList []*Meta, key string) Arrangement {
	a := make(Arrangement)
	for _, m := range metaList {
		if vals, ok := m.GetList(key); ok {
			for _, val := range vals {
				a[val] = append(a[val], m)
			}
		}
	}
	return a
}

func createSimplearrangement(metaList []*Meta, key string) Arrangement {
	a := make(Arrangement)
	for _, m := range metaList {
		if val, ok := m.Get(key); ok && val != "" {
			a[val] = append(a[val], m)
		}
	}
	return a
}

// Counted returns the list of categories, together with the number of
// metadata for each category.
func (a Arrangement) Counted() CountedCategories {
	if len(a) == 0 {
		return nil
	}
	result := make(CountedCategories, 0, len(a))
	for cat, metas := range a {
		result = append(result, CountedCategory{Name: cat, Count: len(metas)})
	}
	return result
}

// CountedCategory contains of a name and the number how much this name occured
// somewhere.
type CountedCategory struct {
	Name  string
	Count int
}

// CountedCategories is the list of CountedCategories.
// Every name must occur only once.
type CountedCategories []CountedCategory

// SortByName sorts the list by the name attribute.
// Since each name must occur only once, two CountedCategories cannot have
// the same name.
func (ccs CountedCategories) SortByName() {
	slices.SortFunc(ccs, func(i, j CountedCategory) int { return strings.Compare(i.Name, j.Name) })
}

// SortByCount sorts the list by the count attribute, descending.
// If two counts are equal, elements are sorted by name.
func (ccs CountedCategories) SortByCount() {
	slices.SortFunc(ccs, func(i, j CountedCategory) int {
		iCount, jCount := i.Count, j.Count
		if iCount > jCount {
			return -1
		}
		if iCount == jCount {
			return strings.Compare(i.Name, j.Name)
		}
		return 1
	})
}

// Categories returns just the category names.
func (ccs CountedCategories) Categories() []string {
	result := make([]string, len(ccs))
	for i, cc := range ccs {
		result[i] = cc.Name
	}
	return result
}
