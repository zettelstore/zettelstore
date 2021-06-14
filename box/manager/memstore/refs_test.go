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

func assertRefs(t *testing.T, i int, got, exp id.Slice) {
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
	t.Parallel()
	testcases := []struct {
		in1, in2   id.Slice
		exp1, exp2 id.Slice
	}{
		{nil, nil, nil, nil},
		{id.Slice{1}, nil, id.Slice{1}, nil},
		{nil, id.Slice{1}, nil, id.Slice{1}},
		{id.Slice{1}, id.Slice{1}, nil, nil},
		{id.Slice{1, 2}, id.Slice{1}, id.Slice{2}, nil},
		{id.Slice{1, 2}, id.Slice{1, 3}, id.Slice{2}, id.Slice{3}},
		{id.Slice{1, 4}, id.Slice{1, 3}, id.Slice{4}, id.Slice{3}},
	}
	for i, tc := range testcases {
		got1, got2 := refsDiff(tc.in1, tc.in2)
		assertRefs(t, i, got1, tc.exp1)
		assertRefs(t, i, got2, tc.exp2)
	}
}

func TestAddRef(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		ref id.Slice
		zid uint
		exp id.Slice
	}{
		{nil, 5, id.Slice{5}},
		{id.Slice{1}, 5, id.Slice{1, 5}},
		{id.Slice{10}, 5, id.Slice{5, 10}},
		{id.Slice{5}, 5, id.Slice{5}},
		{id.Slice{1, 10}, 5, id.Slice{1, 5, 10}},
		{id.Slice{1, 5, 10}, 5, id.Slice{1, 5, 10}},
	}
	for i, tc := range testcases {
		got := addRef(tc.ref, id.Zid(tc.zid))
		assertRefs(t, i, got, tc.exp)
	}
}

func TestRemRefs(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		in1, in2 id.Slice
		exp      id.Slice
	}{
		{nil, nil, nil},
		{nil, id.Slice{}, nil},
		{id.Slice{}, nil, id.Slice{}},
		{id.Slice{}, id.Slice{}, id.Slice{}},
		{id.Slice{1}, id.Slice{5}, id.Slice{1}},
		{id.Slice{10}, id.Slice{5}, id.Slice{10}},
		{id.Slice{1, 5}, id.Slice{5}, id.Slice{1}},
		{id.Slice{5, 10}, id.Slice{5}, id.Slice{10}},
		{id.Slice{1, 10}, id.Slice{5}, id.Slice{1, 10}},
		{id.Slice{1}, id.Slice{2, 5}, id.Slice{1}},
		{id.Slice{10}, id.Slice{2, 5}, id.Slice{10}},
		{id.Slice{1, 5}, id.Slice{2, 5}, id.Slice{1}},
		{id.Slice{5, 10}, id.Slice{2, 5}, id.Slice{10}},
		{id.Slice{1, 2, 5}, id.Slice{2, 5}, id.Slice{1}},
		{id.Slice{2, 5, 10}, id.Slice{2, 5}, id.Slice{10}},
		{id.Slice{1, 10}, id.Slice{2, 5}, id.Slice{1, 10}},
		{id.Slice{1}, id.Slice{5, 9}, id.Slice{1}},
		{id.Slice{10}, id.Slice{5, 9}, id.Slice{10}},
		{id.Slice{1, 5}, id.Slice{5, 9}, id.Slice{1}},
		{id.Slice{5, 10}, id.Slice{5, 9}, id.Slice{10}},
		{id.Slice{1, 5, 9}, id.Slice{5, 9}, id.Slice{1}},
		{id.Slice{5, 9, 10}, id.Slice{5, 9}, id.Slice{10}},
		{id.Slice{1, 10}, id.Slice{5, 9}, id.Slice{1, 10}},
	}
	for i, tc := range testcases {
		got := remRefs(tc.in1, tc.in2)
		assertRefs(t, i, got, tc.exp)
	}
}

func TestRemRef(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		ref id.Slice
		zid uint
		exp id.Slice
	}{
		{nil, 5, nil},
		{id.Slice{}, 5, id.Slice{}},
		{id.Slice{5}, 5, id.Slice{}},
		{id.Slice{1}, 5, id.Slice{1}},
		{id.Slice{10}, 5, id.Slice{10}},
		{id.Slice{1, 5}, 5, id.Slice{1}},
		{id.Slice{5, 10}, 5, id.Slice{10}},
		{id.Slice{1, 5, 10}, 5, id.Slice{1, 10}},
	}
	for i, tc := range testcases {
		got := remRef(tc.ref, id.Zid(tc.zid))
		assertRefs(t, i, got, tc.exp)
	}
}
