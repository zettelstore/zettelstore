//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package notifydir manages the notified directory part of a dirstore.
package notifydir

import (
	"testing"
)

func sameStringSlices(sl1, sl2 []string) bool {
	if len(sl1) != len(sl2) {
		return false
	}
	for i := 0; i < len(sl1); i++ {
		if sl1[i] != sl2[i] {
			return false
		}
	}
	return true
}

func TestMatchValidFileName(t *testing.T) {
	testcases := []struct {
		name string
		exp  []string
	}{
		{"", []string{}},
		{".txt", []string{}},
		{"12345678901234.txt", []string{"12345678901234", ".txt", "txt"}},
		{"12345678901234abc.txt", []string{"12345678901234", ".txt", "txt"}},
		{"12345678901234.abc.txt", []string{"12345678901234", ".txt", "txt"}},
	}

	for i, tc := range testcases {
		got := matchValidFileName(tc.name)
		if len(got) == 0 {
			if len(tc.exp) > 0 {
				t.Errorf("TC=%d, name=%q, exp=%v, got=%v", i, tc.name, tc.exp, got)
			}
		} else {
			if got[0] != tc.name {
				t.Errorf("TC=%d, name=%q, got=%v", i, tc.name, got)
			}
			if !sameStringSlices(got[1:], tc.exp) {
				t.Errorf("TC=%d, name=%q, exp=%v, got=%v", i, tc.name, tc.exp, got)
			}
		}
	}
}
