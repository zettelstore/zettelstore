//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package collect provides functions to collect items from a syntax tree.
package collect

import (
	"zettelstore.de/z/ast"
)

// DivideReferences divides the given list of rederences into zettel, local, and external References.
func DivideReferences(all []*ast.Reference, duplicates bool) (zettel, local, external []*ast.Reference) {
	if len(all) == 0 {
		return nil, nil, nil
	}

	mapZettel := make(map[string]bool)
	mapLocal := make(map[string]bool)
	mapExternal := make(map[string]bool)
	for _, ref := range all {
		if ref.State == ast.RefStateSelf {
			continue
		}
		s := ref.String()
		if ref.IsZettel() {
			if duplicates {
				zettel = append(zettel, ref)
			} else if _, ok := mapZettel[s]; !ok {
				zettel = append(zettel, ref)
				mapZettel[s] = true
			}
		} else if ref.IsExternal() {
			if duplicates {
				external = append(external, ref)
			} else if _, ok := mapExternal[s]; !ok {
				external = append(external, ref)
				mapExternal[s] = true
			}
		} else {
			if duplicates {
				local = append(local, ref)
			} else if _, ok := mapLocal[s]; !ok {
				local = append(local, ref)
				mapLocal[s] = true
			}
		}
	}
	return zettel, local, external
}
