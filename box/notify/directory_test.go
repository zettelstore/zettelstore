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

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
	_ "zettelstore.de/z/parser/blob"       // Allow to use BLOB parser.
	_ "zettelstore.de/z/parser/markdown"   // Allow to use markdown parser.
	_ "zettelstore.de/z/parser/none"       // Allow to use none parser.
	_ "zettelstore.de/z/parser/plain"      // Allow to use plain parser.
	_ "zettelstore.de/z/parser/zettelmark" // Allow to use zettelmark parser.
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

func TestNewExtIsBetter(t *testing.T) {
	extVals := []string{
		// Main Formats
		api.ValueSyntaxZmk, "markdown", "md",
		// Other supported text formats
		"css", "txt", "html", api.ValueSyntaxNone, "mustache", "text", "plain",
		// Supported graphics formats
		"gif", "png", "svg", "jpeg", "jpg",
		// Unsupported syntax values
		"gz", "cpp", "tar", "cppc",
	}
	for oldI, oldExt := range extVals {
		for newI, newExt := range extVals {
			if oldI <= newI {
				continue
			}
			if !newExtIsBetter(oldExt, newExt) {
				t.Errorf("newExtIsBetter(%q, %q) == true, but got false", oldExt, newExt)
			}
			if newExtIsBetter(newExt, oldExt) {
				t.Errorf("newExtIsBetter(%q, %q) == false, but got true", newExt, oldExt)
			}
		}
	}
}
