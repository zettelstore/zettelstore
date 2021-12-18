//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package id_test

import (
	"testing"

	"zettelstore.de/z/domain/id"
)

func TestSetAdd(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		s1, s2 id.Set
		exp    id.Slice
	}{
		{nil, nil, nil},
		{id.NewSet(), nil, nil},
		{id.NewSet(), id.NewSet(), nil},
		{nil, id.NewSet(1), id.Slice{1}},
		{id.NewSet(1), nil, id.Slice{1}},
		{id.NewSet(1), id.NewSet(), id.Slice{1}},
		{id.NewSet(1), id.NewSet(2), id.Slice{1, 2}},
		{id.NewSet(1), id.NewSet(1), id.Slice{1}},
	}
	for i, tc := range testcases {
		sl1 := tc.s1.Sorted()
		sl2 := tc.s2.Sorted()
		got := tc.s1.Add(tc.s2).Sorted()
		if !got.Equal(tc.exp) {
			t.Errorf("%d: %v.Add(%v) should be %v, but got %v", i, sl1, sl2, tc.exp, got)
		}
	}
}

func TestSetSorted(t *testing.T) {
	t.Parallel()
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

func TestSetIntersectOrSet(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		s1, s2 id.Set
		exp    id.Slice
	}{
		{nil, nil, nil},
		{id.NewSet(), nil, nil},
		{nil, id.NewSet(), nil},
		{id.NewSet(), id.NewSet(), nil},
		{id.NewSet(1), nil, nil},
		{nil, id.NewSet(1), id.Slice{1}},
		{id.NewSet(1), id.NewSet(), nil},
		{id.NewSet(), id.NewSet(1), nil},
		{id.NewSet(1), id.NewSet(2), nil},
		{id.NewSet(2), id.NewSet(1), nil},
		{id.NewSet(1), id.NewSet(1), id.Slice{1}},
	}
	for i, tc := range testcases {
		sl1 := tc.s1.Sorted()
		sl2 := tc.s2.Sorted()
		got := tc.s1.IntersectOrSet(tc.s2).Sorted()
		if !got.Equal(tc.exp) {
			t.Errorf("%d: %v.IntersectOrSet(%v) should be %v, but got %v", i, sl1, sl2, tc.exp, got)
		}
	}
}

func TestSetRemove(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		s1, s2 id.Set
		exp    id.Slice
	}{
		{nil, nil, nil},
		{id.NewSet(), nil, nil},
		{id.NewSet(), id.NewSet(), nil},
		{id.NewSet(1), nil, id.Slice{1}},
		{id.NewSet(1), id.NewSet(), id.Slice{1}},
		{id.NewSet(1), id.NewSet(2), id.Slice{1}},
		{id.NewSet(1), id.NewSet(1), id.Slice{}},
	}
	for i, tc := range testcases {
		sl1 := tc.s1.Sorted()
		sl2 := tc.s2.Sorted()
		newS1 := id.NewSet(sl1...)
		newS1.Remove(tc.s2)
		got := newS1.Sorted()
		if !got.Equal(tc.exp) {
			t.Errorf("%d: %v.Remove(%v) should be %v, but got %v", i, sl1, sl2, tc.exp, got)
		}
	}
}
