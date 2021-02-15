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

func TestSort(t *testing.T) {
	zs := id.Slice{9, 4, 6, 1, 7}
	zs.Sort()
	if zs[0] != 1 || zs[1] != 4 || zs[2] != 6 || zs[3] != 7 || zs[4] != 9 {
		t.Errorf("Slice.Sort did not work. Expected %v, got %v", id.Slice{1, 4, 6, 7, 9}, zs)
	}
}

func TestCopy(t *testing.T) {
	var orig id.Slice
	got := orig.Copy()
	if got != nil {
		t.Errorf("Nil copy resulted in %v", got)
	}
	orig = id.Slice{9, 4, 6, 1, 7}
	got = orig.Copy()
	if len(got) != len(orig) || got[0] != 9 || got[1] != 4 || got[2] != 6 || got[3] != 1 || got[4] != 7 {
		t.Errorf("Slice.Copy did not work. Expected %v, got %v", orig, got)
	}
}
func TestString(t *testing.T) {
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
