//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package markdown

import (
	"strings"
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
		var sb strings.Builder
		for _, in := range splitText(tc.text) {
			switch n := in.(type) {
			case *ast.TextNode:
				sb.WriteByte('T')
				sb.WriteString(n.Text)
			case *ast.SpaceNode:
				sb.WriteByte('S')
				sb.WriteString(n.Lexeme)
			default:
				sb.WriteByte('Q')
			}
		}
		got := sb.String()
		if tc.exp != got {
			t.Errorf("TC=%d, text=%q, exp=%q, got=%q", i, tc.text, tc.exp, got)
		}
	}
}
