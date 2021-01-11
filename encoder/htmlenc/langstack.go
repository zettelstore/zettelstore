//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package htmlenc encodes the abstract syntax tree into HTML5.
package htmlenc

import "zettelstore.de/z/ast"

type langStack struct {
	items []string
}

func newLangStack(lang string) langStack {
	items := make([]string, 1, 16)
	items[0] = lang
	return langStack{items}
}

func (s langStack) top() string { return s.items[len(s.items)-1] }

func (s *langStack) pop() { s.items = s.items[0 : len(s.items)-1] }

func (s *langStack) push(attrs *ast.Attributes) {
	if value, ok := attrs.Get("lang"); ok {
		s.items = append(s.items, value)
	} else {
		s.items = append(s.items, s.top())
	}
}
