//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package markdown provides a parser for markdown.
package markdown

import (
	"bytes"
	"testing"

	"zettelstore.de/z/ast"
)

func TestSplitText(t *testing.T) {
	t.Parallel()
	var testcases = []struct {
		text string
		exp  string
	}{
		{"", ""},
		{"abc", "Tabc"},
		{" ", "S "},
		{"abc def", "TabcS Tdef"},
		{"abc def ", "TabcS TdefS "},
		{" abc def ", "S TabcS TdefS "},
	}
	for i, tc := range testcases {
		var buf bytes.Buffer
		for _, in := range splitText(tc.text) {
			switch n := in.(type) {
			case *ast.TextNode:
				buf.WriteByte('T')
				buf.WriteString(n.Text)
			case *ast.SpaceNode:
				buf.WriteByte('S')
				buf.WriteString(n.Lexeme)
			default:
				buf.WriteByte('Q')
			}
		}
		got := buf.String()
		if tc.exp != got {
			t.Errorf("TC=%d, text=%q, exp=%q, got=%q", i, tc.text, tc.exp, got)
		}
	}
}
