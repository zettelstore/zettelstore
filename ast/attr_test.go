//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package ast_test

import (
	"testing"

	"zettelstore.de/z/ast"
)

func TestHasDefault(t *testing.T) {
	t.Parallel()
	attr := ast.Attributes{}
	if attr.HasDefault() {
		t.Error("Should not have default attr")
	}
	attr = ast.Attributes(map[string]string{"-": "value"})
	if !attr.HasDefault() {
		t.Error("Should have default attr")
	}
}

func TestAttrClone(t *testing.T) {
	t.Parallel()
	orig := ast.Attributes{}
	clone := orig.Clone()
	if !clone.IsEmpty() {
		t.Error("Attrs must be empty")
	}

	orig = ast.Attributes(map[string]string{"": "0", "-": "1", "a": "b"})
	clone = orig.Clone()
	if clone[""] != "0" || clone["-"] != "1" || clone["a"] != "b" || len(clone) != len(orig) {
		t.Error("Wrong cloned map")
	}
	clone["a"] = "c"
	if orig["a"] != "b" {
		t.Error("Aliased map")
	}
}
