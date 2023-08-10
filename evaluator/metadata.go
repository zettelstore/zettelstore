//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package evaluator

import (
	"zettelstore.de/z/ast"
	"zettelstore.de/z/zettel/meta"
)

func evaluateMetadata(m *meta.Meta) ast.BlockSlice {
	descrlist := &ast.DescriptionListNode{}
	for _, p := range m.Pairs() {
		descrlist.Descriptions = append(
			descrlist.Descriptions, getMetadataDescription(p.Key, p.Value))
	}
	return ast.BlockSlice{descrlist}
}

func getMetadataDescription(key, value string) ast.Description {
	is := convertMetavalueToInlineSlice(value, meta.Type(key))
	return ast.Description{
		Term:         ast.InlineSlice{&ast.TextNode{Text: key}},
		Descriptions: []ast.DescriptionSlice{{&ast.ParaNode{Inlines: is}}},
	}
}

func convertMetavalueToInlineSlice(value string, dt *meta.DescriptionType) ast.InlineSlice {
	var sliceData []string
	if dt.IsSet {
		sliceData = meta.ListFromValue(value)
		if len(sliceData) == 0 {
			return nil
		}
	} else {
		sliceData = []string{value}
	}
	makeLink := dt == meta.TypeID || dt == meta.TypeIDSet

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
