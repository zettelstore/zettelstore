//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
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

func BenchmarkWalk(b *testing.B) {
	root := ast.CreateBlockListNode(
		&ast.HeadingNode{
			Inlines: ast.CreateInlineListNodeFromWords("A", "Simple", "Heading"),
		},
		&ast.ParaNode{
			Inlines: ast.CreateInlineListNodeFromWords("This", "is", "the", "introduction."),
		},
		&ast.NestedListNode{
			Kind: ast.NestedListUnordered,
			Items: []ast.ItemSlice{
				[]ast.ItemNode{
					&ast.ParaNode{
						Inlines: ast.CreateInlineListNodeFromWords("Item", "1"),
					},
				},
				[]ast.ItemNode{
					&ast.ParaNode{
						Inlines: ast.CreateInlineListNodeFromWords("Item", "2"),
					},
				},
			},
		},
		&ast.ParaNode{
			Inlines: ast.CreateInlineListNodeFromWords("This", "is", "some", "intermediate", "text."),
		},
		ast.CreateParaNode(
			&ast.FormatNode{
				Kind: ast.FormatEmph,
				Attrs: &ast.Attributes{
					Attrs: map[string]string{
						"":      "class",
						"color": "green",
					},
				},
				Inlines: ast.CreateInlineListNodeFromWords("This", "is", "some", "emphasized", "text."),
			},
			&ast.SpaceNode{Lexeme: " "},
			&ast.LinkNode{
				Ref: &ast.Reference{
					Value: "http://zettelstore.de",
				},
				Inlines: ast.CreateInlineListNodeFromWords("URL", "text."),
				OnlyRef: false,
			},
		),
	)
	v := benchVisitor{}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ast.Walk(&v, root)
	}
}

type benchVisitor struct{}

func (bv *benchVisitor) Visit(ast.Node) ast.Visitor { return bv }
