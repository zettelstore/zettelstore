//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package collect_test provides some unit test for collectors.
package collect_test

import (
	"testing"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/collect"
)

func parseRef(s string) *ast.Reference {
	r := ast.ParseReference(s)
	if !r.IsValid() {
		panic(s)
	}
	return r
}

func TestLinks(t *testing.T) {
	t.Parallel()
	zn := &ast.ZettelNode{}
	summary := collect.References(zn)
	if summary.Links != nil || summary.Embeds != nil {
		t.Error("No links/images expected, but got:", summary.Links, "and", summary.Embeds)
	}

	intNode := &ast.LinkNode{Ref: parseRef("01234567890123")}
	para := &ast.ParaNode{
		Inlines: ast.CreateInlineListNode(
			intNode,
			&ast.LinkNode{Ref: parseRef("https://zettelstore.de/z")},
		),
	}
	zn.Ast = &ast.BlockListNode{List: []ast.BlockNode{para}}
	summary = collect.References(zn)
	if summary.Links == nil || summary.Embeds != nil {
		t.Error("Links expected, and no images, but got:", summary.Links, "and", summary.Embeds)
	}

	para.Inlines.Append(intNode)
	summary = collect.References(zn)
	if cnt := len(summary.Links); cnt != 3 {
		t.Error("Link count does not work. Expected: 3, got", summary.Links)
	}
}

func TestEmbed(t *testing.T) {
	t.Parallel()
	zn := &ast.ZettelNode{
		Ast: &ast.BlockListNode{List: []ast.BlockNode{
			&ast.ParaNode{
				Inlines: ast.CreateInlineListNode(
					&ast.EmbedNode{Material: &ast.ReferenceMaterialNode{Ref: parseRef("12345678901234")}},
				),
			},
		}},
	}
	summary := collect.References(zn)
	if summary.Embeds == nil {
		t.Error("Only image expected, but got: ", summary.Embeds)
	}
}
