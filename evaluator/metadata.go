//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package evaluator

import (
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
)

func evaluateMetadata(m *meta.Meta) *ast.BlockListNode {
	descrlist := &ast.DescriptionListNode{}
	for _, p := range m.Pairs(true) {
		descrlist.Descriptions = append(
			descrlist.Descriptions, getMetadataDescription(p.Key, p.Value))
	}
	return &ast.BlockListNode{List: []ast.BlockNode{descrlist}}
}

func getMetadataDescription(key, value string) ast.Description {
	return ast.Description{
		Term: ast.CreateInlineListNode(&ast.TextNode{Text: key}),
		Descriptions: []ast.DescriptionSlice{{
			&ast.ParaNode{
				Inlines: convertMetavalueToInlineList(value, meta.Type(key)),
			}},
		},
	}
}

func convertMetavalueToInlineList(value string, dt *meta.DescriptionType) *ast.InlineListNode {
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
