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

func TestSetOContainsOrNil(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		s   *id.SetO
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
			t.Errorf("%d: %v.ContainsOrNil(%v) == %v, but got %v", i, tc.s, tc.zid, tc.exp, got)
		}
	}
}

func TestSetOAdd(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		s1, s2 *id.SetO
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
		sl1 := tc.s1.SafeSorted()
		sl2 := tc.s2.SafeSorted()
		got := tc.s1.IUnion(tc.s2).SafeSorted()
		if !got.Equal(tc.exp) {
			t.Errorf("%d: %v.Add(%v) should be %v, but got %v", i, sl1, sl2, tc.exp, got)
		}
	}
}

func TestSetOSafeSorted(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		set *id.SetO
		exp id.SliceO
	}{
		{nil, nil},
		{id.NewSetO(), nil},
		{id.NewSetO(9, 4, 6, 1, 7), id.SliceO{1, 4, 6, 7, 9}},
	}
	for i, tc := range testcases {
		got := tc.set.SafeSorted()
		if !got.Equal(tc.exp) {
			t.Errorf("%d: %v.SafeSorted() should be %v, but got %v", i, tc.set, tc.exp, got)
		}
	}
}

func TestSetOIntersectOrSet(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		s1, s2 *id.SetO
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
		sl1 := tc.s1.SafeSorted()
		sl2 := tc.s2.SafeSorted()
		got := tc.s1.IntersectOrSet(tc.s2).SafeSorted()
		if !got.Equal(tc.exp) {
			t.Errorf("%d: %v.IntersectOrSet(%v) should be %v, but got %v", i, sl1, sl2, tc.exp, got)
		}
	}
}

func TestSetOIUnion(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		s1, s2 *id.SetO
		exp    *id.SetO
	}{
		{nil, nil, nil},
		{id.NewSetO(), nil, nil},
		{nil, id.NewSetO(), nil},
		{id.NewSetO(), id.NewSetO(), nil},
		{id.NewSetO(1), nil, id.NewSetO(1)},
		{nil, id.NewSetO(1), id.NewSetO(1)},
		{id.NewSetO(1), id.NewSetO(), id.NewSetO(1)},
		{id.NewSetO(), id.NewSetO(1), id.NewSetO(1)},
		{id.NewSetO(1), id.NewSetO(2), id.NewSetO(1, 2)},
		{id.NewSetO(2), id.NewSetO(1), id.NewSetO(2, 1)},
		{id.NewSetO(1), id.NewSetO(1), id.NewSetO(1)},
		{id.NewSetO(1, 2, 3), id.NewSetO(2, 3, 4), id.NewSetO(1, 2, 3, 4)},
	}
	for i, tc := range testcases {
		s1 := tc.s1.Clone()
		sl1 := s1.SafeSorted()
		sl2 := tc.s2.SafeSorted()
		got := s1.IUnion(tc.s2)
		if !got.Equal(tc.exp) {
			t.Errorf("%d: %v.IUnion(%v) should be %v, but got %v", i, sl1, sl2, tc.exp, got)
		}
	}
}

func TestSetOISubtract(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		s1, s2 *id.SetO
		exp    id.SliceO
	}{
		{nil, nil, nil},
		{id.NewSetO(), nil, nil},
		{nil, id.NewSetO(), nil},
		{id.NewSetO(), id.NewSetO(), nil},
		{id.NewSetO(1), nil, id.SliceO{1}},
		{nil, id.NewSetO(1), nil},
		{id.NewSetO(1), id.NewSetO(), id.SliceO{1}},
		{id.NewSetO(), id.NewSetO(1), nil},
		{id.NewSetO(1), id.NewSetO(2), id.SliceO{1}},
		{id.NewSetO(2), id.NewSetO(1), id.SliceO{2}},
		{id.NewSetO(1), id.NewSetO(1), nil},
		{id.NewSetO(1, 2, 3), id.NewSetO(1), id.SliceO{2, 3}},
		{id.NewSetO(1, 2, 3), id.NewSetO(2), id.SliceO{1, 3}},
		{id.NewSetO(1, 2, 3), id.NewSetO(3), id.SliceO{1, 2}},
		{id.NewSetO(1, 2, 3), id.NewSetO(1, 2), id.SliceO{3}},
		{id.NewSetO(1, 2, 3), id.NewSetO(1, 3), id.SliceO{2}},
		{id.NewSetO(1, 2, 3), id.NewSetO(2, 3), id.SliceO{1}},
	}
	for i, tc := range testcases {
		s1 := tc.s1.Clone()
		sl1 := s1.SafeSorted()
		sl2 := tc.s2.SafeSorted()
		s1.ISubstract(tc.s2)
		got := s1.SafeSorted()
		if !got.Equal(tc.exp) {
			t.Errorf("%d: %v.ISubstract(%v) should be %v, but got %v", i, sl1, sl2, tc.exp, got)
		}
	}
}

func TestSetODiff(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		in1, in2   *id.SetO
		exp1, exp2 *id.SetO
	}{
		{nil, nil, nil, nil},
		{id.NewSetO(1), nil, nil, id.NewSetO(1)},
		{nil, id.NewSetO(1), id.NewSetO(1), nil},
		{id.NewSetO(1), id.NewSetO(1), nil, nil},
		{id.NewSetO(1, 2), id.NewSetO(1), nil, id.NewSetO(2)},
		{id.NewSetO(1), id.NewSetO(1, 2), id.NewSetO(2), nil},
		{id.NewSetO(1, 2), id.NewSetO(1, 3), id.NewSetO(3), id.NewSetO(2)},
		{id.NewSetO(1, 2, 3), id.NewSetO(2, 3, 4), id.NewSetO(4), id.NewSetO(1)},
		{id.NewSetO(2, 3, 4), id.NewSetO(1, 2, 3), id.NewSetO(1), id.NewSetO(4)},
	}
	for i, tc := range testcases {
		gotN, gotO := tc.in1.Diff(tc.in2)
		if !tc.exp1.Equal(gotN) {
			t.Errorf("%d: expected %v, but got: %v", i, tc.exp1, gotN)
		}
		if !tc.exp2.Equal(gotO) {
			t.Errorf("%d: expected %v, but got: %v", i, tc.exp2, gotO)
		}
	}
}

func TestSetORemove(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		s1, s2 *id.SetO
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
		sl1 := tc.s1.SafeSorted()
		sl2 := tc.s2.SafeSorted()
		newS1 := id.NewSetO(sl1...)
		newS1.ISubstract(tc.s2)
		got := newS1.SafeSorted()
		if !got.Equal(tc.exp) {
			t.Errorf("%d: %v.Remove(%v) should be %v, but got %v", i, sl1, sl2, tc.exp, got)
		}
	}
}
func BenchmarkSetO(b *testing.B) {
	s := id.NewSetCapO(b.N)
	for i := range b.N {
		s.Add(id.ZidO(i))
	}
}
