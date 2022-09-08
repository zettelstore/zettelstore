//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package evaluator

import (
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/parser"
)

// ExecuteSearch transforms a list of metadata according to search commands into a AST nested list.
func ExecuteSearch(ml []*meta.Meta, cmds []string) ast.BlockNode {
	kind := ast.NestedListUnordered
	for _, cmd := range cmds {
		if strings.HasPrefix(cmd, "N") {
			kind = ast.NestedListOrdered
			continue
		}
		key := strings.ToLower(cmd)
		mtype := meta.Type(key)
		if mtype == meta.TypeWord {
			return createBlockNodeWord(ml, cmd, kind)
		}
	}
	return createBlockNodeMeta(ml, kind)
}

func createBlockNodeWord(ml []*meta.Meta, key string, kind ast.NestedListKind) ast.BlockNode {
	if len(ml) == 0 {
		return nil
	}
	ccs := meta.CreateArrangement(ml, key).Counted()
	ccs.SortByName()
	items := make([]ast.ItemSlice, 0, len(ccs))
	for _, cat := range ccs {
		items = append(items, ast.ItemSlice{ast.CreateParaNode(&ast.LinkNode{
			Attrs:   nil,
			Ref:     ast.ParseReference(ast.SearchPrefix + key + ":" + cat.Name),
			Inlines: ast.InlineSlice{&ast.TextNode{Text: cat.Name}},
		})})
	}
	return &ast.NestedListNode{
		Kind:  kind,
		Items: items,
		Attrs: nil,
	}
}

func createBlockNodeMeta(ml []*meta.Meta, kind ast.NestedListKind) ast.BlockNode {
	if len(ml) == 0 {
		return nil
	}
	items := make([]ast.ItemSlice, 0, len(ml))
	for _, m := range ml {
		zid := m.Zid.String()
		title, found := m.Get(api.KeyTitle)
		if !found {
			title = zid
		}
		items = append(items, ast.ItemSlice{ast.CreateParaNode(&ast.LinkNode{
			Attrs:   nil,
			Ref:     ast.ParseReference(zid),
			Inlines: parser.ParseMetadataNoLink(title),
		})})
	}
	return &ast.NestedListNode{
		Kind:  kind,
		Items: items,
		Attrs: nil,
	}
}
