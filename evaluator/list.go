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
	"bytes"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/search"
)

// ActionSearch transforms a list of metadata according to search commands into a AST nested list.
func ActionSearch(sea *search.Search, ml []*meta.Meta) ast.BlockNode {
	ap := actionPara{
		sea:  sea,
		ml:   ml,
		kind: ast.NestedListUnordered,
	}
	if actions := sea.Actions(); len(actions) > 0 {
		acts := make([]string, 0, len(actions))
		for _, act := range actions {
			if strings.HasPrefix(act, "N") {
				ap.kind = ast.NestedListOrdered
				continue
			}
			acts = append(acts, act)
		}
		for _, act := range acts {
			key := strings.ToLower(act)
			mtype := meta.Type(key)
			if mtype == meta.TypeWord {
				return ap.createBlockNodeWord(act)
			}
		}
	}
	return ap.createBlockNodeMeta()
}

type actionPara struct {
	sea  *search.Search
	ml   []*meta.Meta
	kind ast.NestedListKind
}

func (ap *actionPara) createBlockNodeWord(key string) ast.BlockNode {
	if len(ap.ml) == 0 {
		return nil
	}
	ccs := meta.CreateArrangement(ap.ml, key).Counted()
	if len(ccs) == 0 {
		return nil
	}

	sea := ap.sea.Clone()
	sea.RemoveActions()
	var buf bytes.Buffer
	buf.WriteString(ast.SearchPrefix)
	sea.Print(&buf)
	if buf.Len() > len(ast.SearchPrefix) {
		buf.WriteByte(' ')
	}
	buf.WriteString(key)
	buf.WriteByte(':')
	bufLen := buf.Len()

	ccs.SortByName()
	items := make([]ast.ItemSlice, 0, len(ccs))
	for _, cat := range ccs {
		buf.WriteString(cat.Name)
		items = append(items, ast.ItemSlice{ast.CreateParaNode(&ast.LinkNode{
			Attrs:   nil,
			Ref:     ast.ParseReference(buf.String()),
			Inlines: ast.InlineSlice{&ast.TextNode{Text: cat.Name}},
		})})
		buf.Truncate(bufLen)
	}
	return &ast.NestedListNode{
		Kind:  ap.kind,
		Items: items,
		Attrs: nil,
	}
}

func (ap *actionPara) createBlockNodeMeta() ast.BlockNode {
	if len(ap.ml) == 0 {
		return nil
	}
	items := make([]ast.ItemSlice, 0, len(ap.ml))
	for _, m := range ap.ml {
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
		Kind:  ap.kind,
		Items: items,
		Attrs: nil,
	}
}
