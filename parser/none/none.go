//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package none provides a none-parser for meta data.
package none

import (
	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
)

func init() {
	parser.Register(&parser.Info{
		Name:          api.ValueSyntaxNone,
		AltNames:      []string{},
		IsTextParser:  false,
		IsImageFormat: false,
		ParseBlocks:   parseBlocks,
		ParseInlines:  parseInlines,
	})
}

func parseBlocks(_ *input.Input, m *meta.Meta, _ string) *ast.BlockListNode {
	descrlist := &ast.DescriptionListNode{}
	for _, p := range m.Pairs(true) {
		descrlist.Descriptions = append(
			descrlist.Descriptions, getDescription(p.Key, p.Value))
	}
	return &ast.BlockListNode{List: []ast.BlockNode{descrlist}}
}

func getDescription(key, value string) ast.Description {
	return ast.Description{
		Term: ast.CreateInlineListNode(&ast.TextNode{Text: key}),
		Descriptions: []ast.DescriptionSlice{{
			&ast.ParaNode{
				Inlines: convertToInlineList(value, meta.Type(key)),
			}},
		},
	}
}

func convertToInlineList(value string, dt *meta.DescriptionType) *ast.InlineListNode {
	var sliceData []string
	if dt.IsSet {
		sliceData = meta.ListFromValue(value)
		if len(sliceData) == 0 {
			return &ast.InlineListNode{}
		}
	} else {
		sliceData = []string{value}
	}
	var makeLink bool
	switch dt {
	case meta.TypeID, meta.TypeIDSet:
		makeLink = true
	}

	result := make([]ast.InlineNode, 0, 2*len(sliceData)-1)
	for i, val := range sliceData {
		if i > 0 {
			result = append(result, &ast.SpaceNode{Lexeme: " "})
		}
		tn := &ast.TextNode{Text: val}
		if makeLink {
			result = append(result, &ast.LinkNode{
				Ref:     ast.ParseReference(val),
				Inlines: ast.CreateInlineListNode(tn),
			})
		} else {
			result = append(result, tn)
		}
	}
	return ast.CreateInlineListNode(result...)
}

func parseInlines(inp *input.Input, _ string) *ast.InlineListNode {
	inp.SkipToEOL()
	return ast.CreateInlineListNode(
		&ast.FormatNode{
			Kind:  ast.FormatSpan,
			Attrs: &ast.Attributes{Attrs: map[string]string{"class": "warning"}},
			Inlines: ast.CreateInlineListNodeFromWords(
				"parser.meta.ParseInlines:", "not", "possible", "("+inp.Src[0:inp.Pos]+")",
			),
		},
	)
}
