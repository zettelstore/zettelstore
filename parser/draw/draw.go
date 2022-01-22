//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package draw provides a parser to create SVG from ASCII drawing
package draw

import (
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
)

func init() {
	parser.Register(&parser.Info{
		Name:          "draw",
		AltNames:      []string{},
		IsTextParser:  true,
		IsImageFormat: false,
		ParseBlocks:   parseBlocks,
		ParseInlines:  parseInlines,
	})
}

func parseBlocks(inp *input.Input, _ *meta.Meta, syntax string) *ast.BlockListNode {
	iln := parseInlines(inp, syntax)
	if iln == nil {
		return nil
	}
	return &ast.BlockListNode{List: []ast.BlockNode{&ast.ParaNode{Inlines: iln}}}
}

func parseInlines(inp *input.Input, syntax string) *ast.InlineListNode {
	svg, err := parseDraw(inp)
	if err != nil || len(svg) == 0 {
		return nil // TODO: besser Fehlertext als AST
	}
	return ast.CreateInlineListNode(&ast.EmbedNode{
		Material: &ast.BLOBMaterialNode{
			Blob:   svg,
			Syntax: "svg",
		},
	})
}

func parseDraw(inp *input.Input) ([]byte, error) {
	canvas, err := NewCanvas(inp.Src[inp.Pos:], 8, true)
	if err != nil {
		return nil, err
	}
	svg := CanvasToSVG(canvas, true, "", 9, 16)
	return svg, nil
}
