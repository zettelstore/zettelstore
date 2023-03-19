//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package collect provides functions to collect items from a syntax tree.
package collect

import (
	"zettelstore.de/z/ast"
	"zettelstore.de/z/strfun"
)

// DivideReferences divides the given list of rederences into zettel, local, and external References.
func DivideReferences(all []*ast.Reference) (zettel, local, external []*ast.Reference) {
	if len(all) == 0 {
		return nil, nil, nil
	}

	mapZettel := make(strfun.Set)
	mapLocal := make(strfun.Set)
	mapExternal := make(strfun.Set)
	for _, ref := range all {
		if ref.State == ast.RefStateSelf {
			continue
		}
		if ref.IsZettel() {
			zettel = appendRefToList(zettel, mapZettel, ref)
		} else if ref.IsExternal() {
			external = appendRefToList(external, mapExternal, ref)
		} else {
			local = appendRefToList(local, mapLocal, ref)
		}
	}
	return zettel, local, external
}

func appendRefToList(reflist []*ast.Reference, refSet strfun.Set, ref *ast.Reference) []*ast.Reference {
	s := ref.String()
	if !refSet.Has(s) {
		reflist = append(reflist, ref)
		refSet.Set(s)
	}
	return reflist
}
