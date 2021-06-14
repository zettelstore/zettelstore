//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package ast provides the abstract syntax tree.
package ast_test

import (
	"testing"

	"zettelstore.de/z/ast"
)

func TestHasDefault(t *testing.T) {
	t.Parallel()
	attr := &ast.Attributes{}
	if attr.HasDefault() {
		t.Error("Should not have default attr")
	}
	attr = &ast.Attributes{Attrs: map[string]string{"-": "value"}}
	if !attr.HasDefault() {
		t.Error("Should have default attr")
	}
}

func TestAttrClone(t *testing.T) {
	t.Parallel()
	orig := &ast.Attributes{}
	clone := orig.Clone()
	if len(clone.Attrs) > 0 {
		t.Error("Attrs must be empty")
	}

	orig = &ast.Attributes{Attrs: map[string]string{"": "0", "-": "1", "a": "b"}}
	clone = orig.Clone()
	m := clone.Attrs
	if m[""] != "0" || m["-"] != "1" || m["a"] != "b" || len(m) != len(orig.Attrs) {
		t.Error("Wrong cloned map")
	}
	m["a"] = "c"
	if orig.Attrs["a"] != "b" {
		t.Error("Aliased map")
	}
}
