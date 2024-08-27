//-----------------------------------------------------------------------------
// Copyright (c) 2024-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2024-present Detlef Stern
//-----------------------------------------------------------------------------

package plain_test

import (
	"testing"

	"t73f.de/r/zsc/input"
	"zettelstore.de/z/encoder/szenc"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/zettel/meta"
)

func TestParseSVG(t *testing.T) {
	testCases := []struct {
		name string
		src  string
		exp  string
	}{
		{"common", " <svg bla", "(INLINE (EMBED-BLOB () \"svg\" \"<svg bla\"))"},
		{"inkscape", "<svg\nbla", "(INLINE (EMBED-BLOB () \"svg\" \"<svg\\nbla\"))"},
		{"selfmade", "<svg>", "(INLINE (EMBED-BLOB () \"svg\" \"<svg>\"))"},
		{"error", "<svgbla", "(INLINE)"},
		{"error-", "<svg-bla", "(INLINE)"},
		{"error#", "<svg2bla", "(INLINE)"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inp := input.NewInput([]byte(tc.src))
			is := parser.ParseInlines(inp, meta.SyntaxSVG)
			trans := szenc.NewTransformer()
			lst := trans.GetSz(&is)
			if got := lst.String(); tc.exp != got {
				t.Errorf("\nexp: %q\ngot: %q", tc.exp, got)
			}
		})
	}
}
