//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package id provides domain specific types, constants, and functions about
// zettel identifier.
package id_test

import (
	"testing"

	"zettelstore.de/z/domain/id"
)

func TestSetSorted(t *testing.T) {
	testcases := []struct {
		set id.Set
		exp id.Slice
	}{
		{nil, nil},
		{id.NewSet(), nil},
		{id.NewSet(9, 4, 6, 1, 7), id.Slice{1, 4, 6, 7, 9}},
	}
	for i, tc := range testcases {
		got := tc.set.Sorted()
		if !got.Equal(tc.exp) {
			t.Errorf("%d: %v.Sorted() should be %v, but got %v", i, tc.set, tc.exp, got)
		}
	}
}

func TestSetIntersection(t *testing.T) {
	testcases := []struct {
		s1, s2 id.Set
		exp    id.Slice
	}{
		{nil, nil, nil},
		{id.NewSet(), nil, nil},
		{id.NewSet(), id.NewSet(), nil},
		{id.NewSet(1), nil, nil},
		{id.NewSet(1), id.NewSet(), nil},
		{id.NewSet(1), id.NewSet(2), nil},
		{id.NewSet(1), id.NewSet(1), id.Slice{1}},
	}
	for i, tc := range testcases {
		sl1 := tc.s1.Sorted()
		sl2 := tc.s2.Sorted()
		got := tc.s1.Intersect(tc.s2).Sorted()
		if !got.Equal(tc.exp) {
			t.Errorf("%d: %v.Intersect(%v) should be %v, but got %v", i, sl1, sl2, tc.exp, got)
		}
		got = id.NewSet(sl2...).Intersect(id.NewSet(sl1...)).Sorted()
		if !got.Equal(tc.exp) {
			t.Errorf("%d: %v.Intersect(%v) should be %v, but got %v", i, sl2, sl1, tc.exp, got)
		}
	}
}
