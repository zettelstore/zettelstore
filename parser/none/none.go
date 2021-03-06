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
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
)

func init() {
	parser.Register(&parser.Info{
		Name:         meta.ValueSyntaxNone,
		AltNames:     []string{},
		ParseBlocks:  parseBlocks,
		ParseInlines: parseInlines,
	})
}

func parseBlocks(inp *input.Input, m *meta.Meta, syntax string) ast.BlockSlice {
	descrlist := &ast.DescriptionListNode{}
	for _, p := range m.Pairs(true) {
		descrlist.Descriptions = append(
			descrlist.Descriptions, getDescription(p.Key, p.Value))
	}
	return ast.BlockSlice{descrlist}
}

func getDescription(key, value string) ast.Description {
	return ast.Description{
		Term: ast.InlineSlice{&ast.TextNode{Text: key}},
		Descriptions: []ast.DescriptionSlice{{
			&ast.ParaNode{
				Inlines: convertToInlineSlice(value, meta.Type(key)),
			}},
		},
	}
}

func convertToInlineSlice(value string, dt *meta.DescriptionType) ast.InlineSlice {
	var sliceData []string
	if dt.IsSet {
		sliceData = meta.ListFromValue(value)
		if len(sliceData) == 0 {
			return ast.InlineSlice{}
		}
	} else {
		sliceData = []string{value}
	}
	var makeLink bool
	switch dt {
	case meta.TypeID, meta.TypeIDSet:
		makeLink = true
	}

	result := make(ast.InlineSlice, 0, 2*len(sliceData)-1)
	for i, val := range sliceData {
		if i > 0 {
			result = append(result, &ast.SpaceNode{Lexeme: " "})
		}
		tn := &ast.TextNode{Text: val}
		if makeLink {
			result = append(result, &ast.LinkNode{
				Ref:     ast.ParseReference(val),
				Inlines: ast.InlineSlice{tn},
			})
		} else {
			result = append(result, tn)
		}
	}
	return result
}

func parseInlines(inp *input.Input, syntax string) ast.InlineSlice {
	inp.SkipToEOL()
	return ast.InlineSlice{
		&ast.FormatNode{
			Kind:  ast.FormatSpan,
			Attrs: &ast.Attributes{Attrs: map[string]string{"class": "warning"}},
			Inlines: ast.InlineSlice{
				&ast.TextNode{Text: "parser.meta.ParseInlines:"},
				&ast.SpaceNode{Lexeme: " "},
				&ast.TextNode{Text: "not"},
				&ast.SpaceNode{Lexeme: " "},
				&ast.TextNode{Text: "possible"},
				&ast.SpaceNode{Lexeme: " "},
				&ast.TextNode{Text: "("},
				&ast.TextNode{Text: inp.Src[0:inp.Pos]},
				&ast.TextNode{Text: ")"},
			},
		},
	}
}
