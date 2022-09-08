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
	}
	return createBlockNodeMeta(ml, kind)
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
