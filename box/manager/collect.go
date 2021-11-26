//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package manager

import (
	"strings"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/box/manager/store"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/strfun"
)

type collectData struct {
	refs  id.Set
	words store.WordSet
	urls  store.WordSet
	itags store.WordSet
}

func (data *collectData) initialize() {
	data.refs = id.NewSet()
	data.words = store.NewWordSet()
	data.urls = store.NewWordSet()
	data.itags = store.NewWordSet()
}

func collectZettelIndexData(zn *ast.ZettelNode, data *collectData) {
	ast.Walk(data, zn.Ast)
}

func collectInlineIndexData(iln *ast.InlineListNode, data *collectData) {
	ast.Walk(data, iln)
}

func (data *collectData) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.VerbatimNode:
		for _, line := range n.Lines {
			data.addText(line)
		}
	case *ast.TextNode:
		data.addText(n.Text)
	case *ast.TagNode:
		data.addText(n.Tag)
		data.itags.Add("#" + strings.ToLower(n.Tag))
	case *ast.LinkNode:
		data.addRef(n.Ref)
	case *ast.EmbedNode:
		if m, ok := n.Material.(*ast.ReferenceMaterialNode); ok {
			data.addRef(m.Ref)
		}
	case *ast.LiteralNode:
		data.addText(n.Text)
	}
	return data
}

func (data *collectData) addText(s string) {
	for _, word := range strfun.NormalizeWords(s) {
		data.words.Add(word)
	}
}

func (data *collectData) addRef(ref *ast.Reference) {
	if ref == nil {
		return
	}
	if ref.IsExternal() {
		data.urls.Add(strings.ToLower(ref.Value))
	}
	if !ref.IsZettel() {
		return
	}
	if zid, err := id.Parse(ref.URL.Path); err == nil {
		data.refs[zid] = true
	}
}
