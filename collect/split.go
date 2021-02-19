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

import "zettelstore.de/z/ast"

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
		if ref.IsZettel() {
			zettel = appendRefToList(zettel, mapZettel, ref, duplicates)
		} else if ref.IsExternal() {
			external = appendRefToList(external, mapExternal, ref, duplicates)
		} else {
			local = appendRefToList(local, mapLocal, ref, duplicates)
		}
	}
	return zettel, local, external
}

func appendRefToList(
	reflist []*ast.Reference,
	refSet map[string]bool,
	ref *ast.Reference,
	duplicates bool,
) []*ast.Reference {
	if duplicates {
		reflist = append(reflist, ref)
	} else {
		s := ref.String()
		if _, ok := refSet[s]; !ok {
			reflist = append(reflist, ref)
			refSet[s] = true
		}
	}

	return reflist
}
