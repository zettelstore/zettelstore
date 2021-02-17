//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
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

// Order of internal reference within the given zettel.
func Order(zn *ast.ZettelNode) (result []*ast.Reference) {
	for _, bn := range zn.Ast {
		if ln, ok := bn.(*ast.NestedListNode); ok {
			switch ln.Code {
			case ast.NestedListOrdered, ast.NestedListUnordered:
				for _, is := range ln.Items {
					if ref := firstItemZettelReference(is); ref != nil {
						result = append(result, ref)
					}
				}
			}
		}
	}
	return result
}

func firstItemZettelReference(is ast.ItemSlice) *ast.Reference {
	for _, in := range is {
		if pn, ok := in.(*ast.ParaNode); ok {
			if ref := firstInlineZettelReference(pn.Inlines); ref != nil {
				return ref
			}
		}
	}
	return nil
}

func firstInlineZettelReference(ins ast.InlineSlice) (result *ast.Reference) {
	for _, inl := range ins {
		switch in := inl.(type) {
		case *ast.LinkNode:
			if ref := in.Ref; ref.IsZettel() {
				return ref
			}
			result = firstInlineZettelReference(in.Inlines)
		case *ast.ImageNode:
			result = firstInlineZettelReference(in.Inlines)
		case *ast.CiteNode:
			result = firstInlineZettelReference(in.Inlines)
		case *ast.FootnoteNode:
			// Ignore references in footnotes
			continue
		case *ast.FormatNode:
			result = firstInlineZettelReference(in.Inlines)
		default:
			continue
		}
		if result != nil {
			return result
		}
	}
	return nil
}
