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

func TestSliceOSort(t *testing.T) {
	t.Parallel()
	zs := id.SliceO{9, 4, 6, 1, 7}
	zs.Sort()
	exp := id.SliceO{1, 4, 6, 7, 9}
	if !zs.Equal(exp) {
		t.Errorf("Slice.Sort did not work. Expected %v, got %v", exp, zs)
	}
}

func TestSliceOCopy(t *testing.T) {
	t.Parallel()
	var orig id.SliceO
	got := orig.Clone()
	if got != nil {
		t.Errorf("Nil copy resulted in %v", got)
	}
	orig = id.SliceO{9, 4, 6, 1, 7}
	got = orig.Clone()
	if !orig.Equal(got) {
		t.Errorf("Slice.Copy did not work. Expected %v, got %v", orig, got)
	}
}

func TestSliceOEqual(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		s1, s2 id.SliceO
		exp    bool
	}{
		{nil, nil, true},
		{nil, id.SliceO{}, true},
		{nil, id.SliceO{1}, false},
		{id.SliceO{1}, id.SliceO{1}, true},
		{id.SliceO{1}, id.SliceO{2}, false},
		{id.SliceO{1, 2}, id.SliceO{2, 1}, false},
		{id.SliceO{1, 2}, id.SliceO{1, 2}, true},
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

func TestSlice0MetaString(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		in  id.SliceO
		exp string
	}{
		{nil, ""},
		{id.SliceO{}, ""},
		{id.SliceO{1}, "00000000000001"},
		{id.SliceO{1, 2}, "00000000000001 00000000000002"},
	}
	for i, tc := range testcases {
		got := tc.in.MetaString()
		if got != tc.exp {
			t.Errorf("%d/%v: expected %q, but got %q", i, tc.in, tc.exp, got)
		}
	}
}
