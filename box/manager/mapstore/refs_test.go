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

package mapstore

import (
	"testing"

	"zettelstore.de/z/zettel/id"
)

func assertRefs(t *testing.T, i int, got, exp id.SliceO) {
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
		if got := got[p]; got != id.ZidO(n) {
			t.Errorf("%d: pos %d: expected %d, but got %d", i, p, n, got)
		}
	}
}

func TestRefsDiff(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		in1, in2   id.SliceO
		exp1, exp2 id.SliceO
	}{
		{nil, nil, nil, nil},
		{id.SliceO{1}, nil, id.SliceO{1}, nil},
		{nil, id.SliceO{1}, nil, id.SliceO{1}},
		{id.SliceO{1}, id.SliceO{1}, nil, nil},
		{id.SliceO{1, 2}, id.SliceO{1}, id.SliceO{2}, nil},
		{id.SliceO{1, 2}, id.SliceO{1, 3}, id.SliceO{2}, id.SliceO{3}},
		{id.SliceO{1, 4}, id.SliceO{1, 3}, id.SliceO{4}, id.SliceO{3}},
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
		ref id.SliceO
		zid uint
		exp id.SliceO
	}{
		{nil, 5, id.SliceO{5}},
		{id.SliceO{1}, 5, id.SliceO{1, 5}},
		{id.SliceO{10}, 5, id.SliceO{5, 10}},
		{id.SliceO{5}, 5, id.SliceO{5}},
		{id.SliceO{1, 10}, 5, id.SliceO{1, 5, 10}},
		{id.SliceO{1, 5, 10}, 5, id.SliceO{1, 5, 10}},
	}
	for i, tc := range testcases {
		got := addRef(tc.ref, id.ZidO(tc.zid))
		assertRefs(t, i, got, tc.exp)
	}
}

func TestRemRefs(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		in1, in2 id.SliceO
		exp      id.SliceO
	}{
		{nil, nil, nil},
		{nil, id.SliceO{}, nil},
		{id.SliceO{}, nil, id.SliceO{}},
		{id.SliceO{}, id.SliceO{}, id.SliceO{}},
		{id.SliceO{1}, id.SliceO{5}, id.SliceO{1}},
		{id.SliceO{10}, id.SliceO{5}, id.SliceO{10}},
		{id.SliceO{1, 5}, id.SliceO{5}, id.SliceO{1}},
		{id.SliceO{5, 10}, id.SliceO{5}, id.SliceO{10}},
		{id.SliceO{1, 10}, id.SliceO{5}, id.SliceO{1, 10}},
		{id.SliceO{1}, id.SliceO{2, 5}, id.SliceO{1}},
		{id.SliceO{10}, id.SliceO{2, 5}, id.SliceO{10}},
		{id.SliceO{1, 5}, id.SliceO{2, 5}, id.SliceO{1}},
		{id.SliceO{5, 10}, id.SliceO{2, 5}, id.SliceO{10}},
		{id.SliceO{1, 2, 5}, id.SliceO{2, 5}, id.SliceO{1}},
		{id.SliceO{2, 5, 10}, id.SliceO{2, 5}, id.SliceO{10}},
		{id.SliceO{1, 10}, id.SliceO{2, 5}, id.SliceO{1, 10}},
		{id.SliceO{1}, id.SliceO{5, 9}, id.SliceO{1}},
		{id.SliceO{10}, id.SliceO{5, 9}, id.SliceO{10}},
		{id.SliceO{1, 5}, id.SliceO{5, 9}, id.SliceO{1}},
		{id.SliceO{5, 10}, id.SliceO{5, 9}, id.SliceO{10}},
		{id.SliceO{1, 5, 9}, id.SliceO{5, 9}, id.SliceO{1}},
		{id.SliceO{5, 9, 10}, id.SliceO{5, 9}, id.SliceO{10}},
		{id.SliceO{1, 10}, id.SliceO{5, 9}, id.SliceO{1, 10}},
	}
	for i, tc := range testcases {
		got := remRefs(tc.in1, tc.in2)
		assertRefs(t, i, got, tc.exp)
	}
}

func TestRemRef(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		ref id.SliceO
		zid uint
		exp id.SliceO
	}{
		{nil, 5, nil},
		{id.SliceO{}, 5, id.SliceO{}},
		{id.SliceO{5}, 5, id.SliceO{}},
		{id.SliceO{1}, 5, id.SliceO{1}},
		{id.SliceO{10}, 5, id.SliceO{10}},
		{id.SliceO{1, 5}, 5, id.SliceO{1}},
		{id.SliceO{5, 10}, 5, id.SliceO{10}},
		{id.SliceO{1, 5, 10}, 5, id.SliceO{1, 10}},
	}
	for i, tc := range testcases {
		got := remRef(tc.ref, id.ZidO(tc.zid))
		assertRefs(t, i, got, tc.exp)
	}
}
