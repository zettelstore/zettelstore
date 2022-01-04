//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package notify

import (
	"testing"

	"zettelstore.de/z/domain/id"
)

func TestSeekZidExt(t *testing.T) {
	testcases := []struct {
		name string
		zid  id.Zid
		ext  string
	}{
		{"", id.Invalid, ""},
		{"12345678901234.ext", id.Zid(12345678901234), "ext"},
		{"12345678901234 abc.ext", id.Zid(12345678901234), "ext"},
		{"12345678901234", id.Zid(12345678901234), ""},
		{"12345678901234 def", id.Zid(12345678901234), ""},
	}
	for _, tc := range testcases {
		gotZid, gotExt := seekZidExt(tc.name)
		if gotZid != tc.zid {
			t.Errorf("seekZidExt(%q) == %v, but got %v", tc.name, tc.zid, gotZid)
		} else if gotExt != tc.ext {
			t.Errorf("seekZidExt(%q) == %q, but got %q", tc.name, tc.ext, gotExt)
		}
	}
}
