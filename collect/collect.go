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

// Summary stores the relevant parts of the syntax tree
type Summary struct {
	Links  []*ast.Reference // list of all referenced links
	Images []*ast.Reference // list of all referenced images
	Cites  []*ast.CiteNode  // list of all referenced citations
}

// References returns all references mentioned in the given zettel. This also
// includes references to images.
func References(zn *ast.ZettelNode) Summary {
	var s Summary
	ast.WalkBlockSlice(&s, zn.Ast)
	return s
}

func (s *Summary) Visit(node ast.Node) ast.WalkVisitor {
	switch n := node.(type) {
	case *ast.LinkNode:
		s.Links = append(s.Links, n.Ref)
	case *ast.ImageNode:
		if n.Ref != nil {
			s.Images = append(s.Images, n.Ref)
		}
	case *ast.CiteNode:
		s.Cites = append(s.Cites, n)
	}
	return s
}
