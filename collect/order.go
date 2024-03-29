//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2021-present Detlef Stern
//-----------------------------------------------------------------------------

// Package collect provides functions to collect items from a syntax tree.
package collect

import "zettelstore.de/z/ast"

// Order of internal reference within the given zettel.
func Order(zn *ast.ZettelNode) (result []*ast.Reference) {
	for _, bn := range zn.Ast {
		ln, ok := bn.(*ast.NestedListNode)
		if !ok {
			continue
		}
		switch ln.Kind {
		case ast.NestedListOrdered, ast.NestedListUnordered:
			for _, is := range ln.Items {
				if ref := firstItemZettelReference(is); ref != nil {
					result = append(result, ref)
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

func firstInlineZettelReference(is ast.InlineSlice) (result *ast.Reference) {
	for _, inl := range is {
		switch in := inl.(type) {
		case *ast.LinkNode:
			if ref := in.Ref; ref.IsZettel() {
				return ref
			}
			result = firstInlineZettelReference(in.Inlines)
		case *ast.EmbedRefNode:
			result = firstInlineZettelReference(in.Inlines)
		case *ast.EmbedBLOBNode:
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
