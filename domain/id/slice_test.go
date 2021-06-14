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

func TestSliceSort(t *testing.T) {
	t.Parallel()
	zs := id.Slice{9, 4, 6, 1, 7}
	zs.Sort()
	exp := id.Slice{1, 4, 6, 7, 9}
	if !zs.Equal(exp) {
		t.Errorf("Slice.Sort did not work. Expected %v, got %v", exp, zs)
	}
}

func TestCopy(t *testing.T) {
	t.Parallel()
	var orig id.Slice
	got := orig.Copy()
	if got != nil {
		t.Errorf("Nil copy resulted in %v", got)
	}
	orig = id.Slice{9, 4, 6, 1, 7}
	got = orig.Copy()
	if !orig.Equal(got) {
		t.Errorf("Slice.Copy did not work. Expected %v, got %v", orig, got)
	}
}

func TestSliceEqual(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		s1, s2 id.Slice
		exp    bool
	}{
		{nil, nil, true},
		{nil, id.Slice{}, true},
		{nil, id.Slice{1}, false},
		{id.Slice{1}, id.Slice{1}, true},
		{id.Slice{1}, id.Slice{2}, false},
		{id.Slice{1, 2}, id.Slice{2, 1}, false},
		{id.Slice{1, 2}, id.Slice{1, 2}, true},
	}
	for i, tc := range testcases {
		got := tc.s1.Equal(tc.s2)
		if got != tc.exp {
			t.Errorf("%d/%v.Equal(%v)==%v, but got %v", i, tc.s1, tc.s2, tc.exp, got)
		}
		got = tc.s2.Equal(tc.s1)
		if got != tc.exp {
			t.Errorf("%d/%v.Equal(%v)==%v, but got %v", i, tc.s2, tc.s1, tc.exp, got)
		}
	}
}

func TestSliceString(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		in  id.Slice
		exp string
	}{
		{nil, ""},
		{id.Slice{}, ""},
		{id.Slice{1}, "00000000000001"},
		{id.Slice{1, 2}, "00000000000001 00000000000002"},
	}
	for i, tc := range testcases {
		got := tc.in.String()
		if got != tc.exp {
			t.Errorf("%d/%v: expected %q, but got %q", i, tc.in, tc.exp, got)
		}
	}
}
