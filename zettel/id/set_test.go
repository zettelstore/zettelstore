//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2021-present Detlef Stern
//-----------------------------------------------------------------------------

package id_test

import (
	"testing"

	"zettelstore.de/z/zettel/id"
)

func TestSetContainsO(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		s   id.SetO
		zid id.ZidO
		exp bool
	}{
		{nil, id.InvalidO, true},
		{nil, 14, true},
		{id.NewSetO(), id.InvalidO, false},
		{id.NewSetO(), 1, false},
		{id.NewSetO(), id.InvalidO, false},
		{id.NewSetO(1), 1, true},
	}
	for i, tc := range testcases {
		got := tc.s.ContainsOrNil(tc.zid)
		if got != tc.exp {
			t.Errorf("%d: %v.Contains(%v) == %v, but got %v", i, tc.s, tc.zid, tc.exp, got)
		}
	}
}

func TestSetAddO(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		s1, s2 id.SetO
		exp    id.SliceO
	}{
		{nil, nil, nil},
		{id.NewSetO(), nil, nil},
		{id.NewSetO(), id.NewSetO(), nil},
		{nil, id.NewSetO(1), id.SliceO{1}},
		{id.NewSetO(1), nil, id.SliceO{1}},
		{id.NewSetO(1), id.NewSetO(), id.SliceO{1}},
		{id.NewSetO(1), id.NewSetO(2), id.SliceO{1, 2}},
		{id.NewSetO(1), id.NewSetO(1), id.SliceO{1}},
	}
	for i, tc := range testcases {
		sl1 := tc.s1.Sorted()
		sl2 := tc.s2.Sorted()
		got := tc.s1.Copy(tc.s2).Sorted()
		if !got.Equal(tc.exp) {
			t.Errorf("%d: %v.Add(%v) should be %v, but got %v", i, sl1, sl2, tc.exp, got)
		}
	}
}

func TestSetSortedO(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		set id.SetO
		exp id.SliceO
	}{
		{nil, nil},
		{id.NewSetO(), nil},
		{id.NewSetO(9, 4, 6, 1, 7), id.SliceO{1, 4, 6, 7, 9}},
	}
	for i, tc := range testcases {
		got := tc.set.Sorted()
		if !got.Equal(tc.exp) {
			t.Errorf("%d: %v.Sorted() should be %v, but got %v", i, tc.set, tc.exp, got)
		}
	}
}

func TestSetIntersectOrSetO(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		s1, s2 id.SetO
		exp    id.SliceO
	}{
		{nil, nil, nil},
		{id.NewSetO(), nil, nil},
		{nil, id.NewSetO(), nil},
		{id.NewSetO(), id.NewSetO(), nil},
		{id.NewSetO(1), nil, nil},
		{nil, id.NewSetO(1), id.SliceO{1}},
		{id.NewSetO(1), id.NewSetO(), nil},
		{id.NewSetO(), id.NewSetO(1), nil},
		{id.NewSetO(1), id.NewSetO(2), nil},
		{id.NewSetO(2), id.NewSetO(1), nil},
		{id.NewSetO(1), id.NewSetO(1), id.SliceO{1}},
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

func TestSetRemoveO(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		s1, s2 id.SetO
		exp    id.SliceO
	}{
		{nil, nil, nil},
		{id.NewSetO(), nil, nil},
		{id.NewSetO(), id.NewSetO(), nil},
		{id.NewSetO(1), nil, id.SliceO{1}},
		{id.NewSetO(1), id.NewSetO(), id.SliceO{1}},
		{id.NewSetO(1), id.NewSetO(2), id.SliceO{1}},
		{id.NewSetO(1), id.NewSetO(1), id.SliceO{}},
	}
	for i, tc := range testcases {
		sl1 := tc.s1.Sorted()
		sl2 := tc.s2.Sorted()
		newS1 := id.NewSetO(sl1...)
		newS1.Substract(tc.s2)
		got := newS1.Sorted()
		if !got.Equal(tc.exp) {
			t.Errorf("%d: %v.Remove(%v) should be %v, but got %v", i, sl1, sl2, tc.exp, got)
		}
	}
}

//	func BenchmarkSet(b *testing.B) {
//		s := id.Set{}
//		for range b.N {
//			s[id.Zid(i)] = true
//		}
//	}
func BenchmarkSetO(b *testing.B) {
	s := id.SetO{}
	for i := range b.N {
		s[id.ZidO(i)] = struct{}{}
	}
}
