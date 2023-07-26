//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
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

	"zettelstore.de/client.fossil/attrs"
	"zettelstore.de/z/ast"
)

func BenchmarkWalk(b *testing.B) {
	root := ast.BlockSlice{
		&ast.HeadingNode{
			Inlines: ast.CreateInlineSliceFromWords("A", "Simple", "Heading"),
		},
		&ast.ParaNode{
			Inlines: ast.CreateInlineSliceFromWords("This", "is", "the", "introduction."),
		},
		&ast.NestedListNode{
			Kind: ast.NestedListUnordered,
			Items: []ast.ItemSlice{
				[]ast.ItemNode{
					&ast.ParaNode{
						Inlines: ast.CreateInlineSliceFromWords("Item", "1"),
					},
				},
				[]ast.ItemNode{
					&ast.ParaNode{
						Inlines: ast.CreateInlineSliceFromWords("Item", "2"),
					},
				},
			},
		},
		&ast.ParaNode{
			Inlines: ast.CreateInlineSliceFromWords("This", "is", "some", "intermediate", "text."),
		},
		ast.CreateParaNode(
			&ast.FormatNode{
				Kind: ast.FormatEmph,
				Attrs: attrs.Attributes(map[string]string{
					"":      "class",
					"color": "green",
				}),
				Inlines: ast.CreateInlineSliceFromWords("This", "is", "some", "emphasized", "text."),
			},
			&ast.SpaceNode{Lexeme: " "},
			&ast.LinkNode{
				Ref:     &ast.Reference{Value: "http://zettelstore.de"},
				Inlines: ast.CreateInlineSliceFromWords("URL", "text."),
			},
		),
	}
	v := benchVisitor{}
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ast.Walk(&v, &root)
	}
}

type benchVisitor struct{}

func (bv *benchVisitor) Visit(ast.Node) ast.Visitor { return bv }
