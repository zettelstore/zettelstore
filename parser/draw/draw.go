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

func parseBlocks(inp *input.Input, m *meta.Meta, _ string) *ast.BlockListNode {
	font := m.GetDefault("font", "")
	scaleX := m.GetNumber("x-scale", 8)
	scaleY := m.GetNumber("y-scale", 16)
	iln := parseDraw(inp, font, scaleX, scaleY)
	if iln == nil {
		return nil
	}
	return &ast.BlockListNode{List: []ast.BlockNode{&ast.ParaNode{Inlines: iln}}}
}

func parseInlines(inp *input.Input, _ string) *ast.InlineListNode {
	return parseDraw(inp, "", 8, 16)
}

func parseDraw(inp *input.Input, font string, scaleX, scaleY int) *ast.InlineListNode {
	canvas, err := NewCanvas(inp.Src[inp.Pos:], 8)
	if err != nil {
		return nil // TODO: Fehlertext err.Error()
	}
	svg := CanvasToSVG(canvas, font, scaleX, scaleY)
	if len(svg) == 0 {
		return nil // TODO: Fehlertext "no image"
	}
	return ast.CreateInlineListNode(&ast.EmbedNode{
		Material: &ast.BLOBMaterialNode{
			Blob:   svg,
			Syntax: "svg",
		},
	})
}
