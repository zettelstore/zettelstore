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

import (
	"testing"

	"zettelstore.de/z/ast"
)

func TestStackSimple(t *testing.T) {
	exp := "de"
	s := newLangStack(exp)
	if got := s.top(); got != exp {
		t.Errorf("Init: expected %q, but got %q", exp, got)
		return
	}

	a := &ast.Attributes{}
	s.push(a)
	if got := s.top(); exp != got {
		t.Errorf("Empty push: expected %q, but got %q", exp, got)
	}

	exp2 := "en"
	a = a.Set("lang", exp2)
	s.push(a)
	if got := s.top(); exp2 != got {
		t.Errorf("Full push: expected %q, but got %q", exp2, got)
	}

	s.pop()
	if got := s.top(); exp != got {
		t.Errorf("pop: expected %q, but got %q", exp, got)
	}
}
