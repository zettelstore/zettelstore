//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package input_test provides some unit-tests for reading data.
package input_test

import (
	"testing"

	"zettelstore.de/z/input"
)

func TestEatEOL(t *testing.T) {
	t.Parallel()
	inp := input.NewInput(nil)
	inp.EatEOL()
	if inp.Ch != input.EOS {
		t.Errorf("No EOS found: %q", inp.Ch)
	}
	if inp.Pos != 0 {
		t.Errorf("Pos != 0: %d", inp.Pos)
	}

	inp = input.NewInput([]byte("ABC"))
	if inp.Ch != 'A' {
		t.Errorf("First ch != 'A', got %q", inp.Ch)
	}
	inp.EatEOL()
	if inp.Ch != 'A' {
		t.Errorf("First ch != 'A', got %q", inp.Ch)
	}
}

func TestAccept(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		accept string
		src    string
		acc    bool
		exp    rune
	}{
		{"", "", false, input.EOS},
		{"AB", "abc", false, 'a'},
		{"AB", "ABC", true, 'C'},
		{"AB", "AB", true, input.EOS},
		{"AB", "A", false, 'A'},
	}
	for i, tc := range testcases {
		inp := input.NewInput([]byte(tc.src))
		acc := inp.Accept(tc.accept)
		if acc != tc.acc {
			t.Errorf("%d: %q.Accept(%q) == %v, but got %v", i, tc.src, tc.accept, tc.acc, acc)
		}
		if got := inp.Ch; tc.exp != got {
			t.Errorf("%d: %q.Accept(%q) should result in run %v, but got %v", i, tc.src, tc.accept, tc.exp, got)
		}
	}
}
