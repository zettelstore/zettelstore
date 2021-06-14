//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
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
	inp := input.NewInput("")
	inp.EatEOL()
	if inp.Ch != input.EOS {
		t.Errorf("No EOS found: %q", inp.Ch)
	}
	if inp.Pos != 0 {
		t.Errorf("Pos != 0: %d", inp.Pos)
	}

	inp = input.NewInput("ABC")
	if inp.Ch != 'A' {
		t.Errorf("First ch != 'A', got %q", inp.Ch)
	}
	inp.EatEOL()
	if inp.Ch != 'A' {
		t.Errorf("First ch != 'A', got %q", inp.Ch)
	}
}

func TestScanEntity(t *testing.T) {
	t.Parallel()
	var testcases = []struct {
		text string
		exp  string
	}{
		{"", ""},
		{"a", ""},
		{"&amp;", "&"},
		{"&#9;", "\t"},
		{"&quot;", "\""},
	}
	for id, tc := range testcases {
		inp := input.NewInput(tc.text)
		got, ok := inp.ScanEntity()
		if !ok {
			if tc.exp != "" {
				t.Errorf("ID=%d, text=%q: expected error, but got %q", id, tc.text, got)
			}
			if inp.Pos != 0 {
				t.Errorf("ID=%d, text=%q: input position advances to %d", id, tc.text, inp.Pos)
			}
			continue
		}
		if tc.exp != got {
			t.Errorf("ID=%d, text=%q: expected %q, but got %q", id, tc.text, tc.exp, got)
		}
	}
}
