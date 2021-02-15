//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package memstore stored the index in main memory.
package memstore

import (
	"testing"

	"zettelstore.de/z/domain/id"
)

func numsToRefs(nums []uint) id.Slice {
	if nums == nil {
		return nil
	}
	refs := make(id.Slice, 0, len(nums))
	for _, n := range nums {
		refs = append(refs, id.Zid(n))
	}
	return refs
}

func assertRefs(t *testing.T, i int, got id.Slice, exp []uint) {
	t.Helper()
	if got == nil && exp != nil {
		t.Errorf("%d: got nil, but expected %v", i, exp)
		return
	}
	if got != nil && exp == nil {
		t.Errorf("%d: expected nil, but got %v", i, got)
		return
	}
	if len(got) != len(exp) {
		t.Errorf("%d: expected len(%v)==%d, but got len(%v)==%d", i, exp, len(exp), got, len(got))
		return
	}
	for p, n := range exp {
		if got := got[p]; got != id.Zid(n) {
			t.Errorf("%d: pos %d: expected %d, but got %d", i, p, n, got)
		}
	}
}

func TestRefsDiff(t *testing.T) {
	testcases := []struct {
		in1, in2   []uint
		exp1, exp2 []uint
	}{
		{nil, nil, nil, nil},
		{[]uint{1}, nil, []uint{1}, nil},
		{nil, []uint{1}, nil, []uint{1}},
		{[]uint{1}, []uint{1}, nil, nil},
		{[]uint{1, 2}, []uint{1}, []uint{2}, nil},
		{[]uint{1, 2}, []uint{1, 3}, []uint{2}, []uint{3}},
		{[]uint{1, 4}, []uint{1, 3}, []uint{4}, []uint{3}},
	}
	for i, tc := range testcases {
		got1, got2 := refsDiff(numsToRefs(tc.in1), numsToRefs(tc.in2))
		assertRefs(t, i, got1, tc.exp1)
		assertRefs(t, i, got2, tc.exp2)
	}
}

func TestAddRef(t *testing.T) {
	testcases := []struct {
		ref []uint
		zid uint
		exp []uint
	}{
		{nil, 5, []uint{5}},
		{[]uint{1}, 5, []uint{1, 5}},
		{[]uint{10}, 5, []uint{5, 10}},
		{[]uint{5}, 5, []uint{5}},
		{[]uint{1, 10}, 5, []uint{1, 5, 10}},
		{[]uint{1, 5, 10}, 5, []uint{1, 5, 10}},
	}
	for i, tc := range testcases {
		got := addRef(numsToRefs(tc.ref), id.Zid(tc.zid))
		assertRefs(t, i, got, tc.exp)
	}
}

func TestRemRefs(t *testing.T) {
	testcases := []struct {
		in1, in2 []uint
		exp      []uint
	}{
		{nil, nil, nil},
		{nil, []uint{}, nil},
		{[]uint{}, nil, []uint{}},
		{[]uint{}, []uint{}, []uint{}},
		{[]uint{1}, []uint{5}, []uint{1}},
		{[]uint{10}, []uint{5}, []uint{10}},
		{[]uint{1, 5}, []uint{5}, []uint{1}},
		{[]uint{5, 10}, []uint{5}, []uint{10}},
		{[]uint{1, 10}, []uint{5}, []uint{1, 10}},
		{[]uint{1}, []uint{2, 5}, []uint{1}},
		{[]uint{10}, []uint{2, 5}, []uint{10}},
		{[]uint{1, 5}, []uint{2, 5}, []uint{1}},
		{[]uint{5, 10}, []uint{2, 5}, []uint{10}},
		{[]uint{1, 2, 5}, []uint{2, 5}, []uint{1}},
		{[]uint{2, 5, 10}, []uint{2, 5}, []uint{10}},
		{[]uint{1, 10}, []uint{2, 5}, []uint{1, 10}},
		{[]uint{1}, []uint{5, 9}, []uint{1}},
		{[]uint{10}, []uint{5, 9}, []uint{10}},
		{[]uint{1, 5}, []uint{5, 9}, []uint{1}},
		{[]uint{5, 10}, []uint{5, 9}, []uint{10}},
		{[]uint{1, 5, 9}, []uint{5, 9}, []uint{1}},
		{[]uint{5, 9, 10}, []uint{5, 9}, []uint{10}},
		{[]uint{1, 10}, []uint{5, 9}, []uint{1, 10}},
	}
	for i, tc := range testcases {
		got := remRefs(numsToRefs(tc.in1), numsToRefs(tc.in2))
		assertRefs(t, i, got, tc.exp)
	}
}

func TestRemRef(t *testing.T) {
	testcases := []struct {
		ref []uint
		zid uint
		exp []uint
	}{
		{nil, 5, nil},
		{[]uint{}, 5, []uint{}},
		{[]uint{1}, 5, []uint{1}},
		{[]uint{10}, 5, []uint{10}},
		{[]uint{1, 5}, 5, []uint{1}},
		{[]uint{5, 10}, 5, []uint{10}},
		{[]uint{1, 5, 10}, 5, []uint{1, 10}},
	}
	for i, tc := range testcases {
		got := remRef(numsToRefs(tc.ref), id.Zid(tc.zid))
		assertRefs(t, i, got, tc.exp)
	}
}
