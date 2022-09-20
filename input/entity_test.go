//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package input_test

import (
	"testing"

	"zettelstore.de/z/input"
)

func TestScanEntity(t *testing.T) {
	t.Parallel()
	var testcases = []struct {
		text string
		exp  string
	}{
		{"", ""},
		{"a", ""},
		{"&amp;", "&"},
		{"&#33;", "!"},
		{"&#x33;", "3"},
		{"&quot;", "\""},
	}
	for id, tc := range testcases {
		inp := input.NewInput([]byte(tc.text))
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

func TestScanIllegalEntity(t *testing.T) {
	t.Parallel()
	testcases := []string{"", "a", "& Input &rarr;", "&#9;", "&#x1f;"}
	for i, tc := range testcases {
		inp := input.NewInput([]byte(tc))
		got, ok := inp.ScanEntity()
		if ok {
			t.Errorf("%d: scanning %q was unexpected successful, got %q", i, tc, got)
			continue
		}
	}
}
