//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package draw provides a parser to create SVG from ASCII drawing
package draw

import (
	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
)

func init() {
	parser.Register(&parser.Info{
		Name:          api.ValueSyntaxDraw,
		AltNames:      []string{},
		IsTextParser:  true,
		IsImageFormat: false,
		ParseBlocks:   parseBlocks,
		ParseInlines:  parseInlines,
	})
}

const (
	defaultTabSize = 8
	defaultFont    = ""
	defaultScaleX  = 10
	defaultScaleY  = 20
)

func parseBlocks(inp *input.Input, m *meta.Meta, _ string) ast.BlockListNode {
	font := m.GetDefault("font", defaultFont)
	scaleX := m.GetNumber("x-scale", defaultScaleX)
	scaleY := m.GetNumber("y-scale", defaultScaleY)
	canvas, err := newCanvas(inp.Src[inp.Pos:], defaultTabSize)
	if err != nil {
		iln := canvasErrMsg(err)
		return ast.CreateBlockListNode(ast.CreateParaNode(&iln))
	}
	if scaleX < 1 || 1000000 < scaleX {
		scaleX = defaultScaleX
	}
	if scaleY < 1 || 1000000 < scaleY {
		scaleY = defaultScaleY
	}
	svg := canvasToSVG(canvas, font, int(scaleX), int(scaleY))
	if len(svg) == 0 {
		iln := noSVGErrMsg()
		return ast.CreateBlockListNode(ast.CreateParaNode(&iln))
	}
	return ast.CreateBlockListNode(&ast.BLOBNode{
		Title:  "",
		Syntax: api.ValueSyntaxSVG,
		Blob:   svg,
	})
}

func parseInlines(inp *input.Input, _ string) ast.InlineListNode {
	canvas, err := newCanvas(inp.Src[inp.Pos:], defaultTabSize)
	if err != nil {
		return canvasErrMsg(err)
	}
	svg := canvasToSVG(canvas, defaultFont, defaultScaleX, defaultScaleY)
	if len(svg) == 0 {
		return noSVGErrMsg()
	}
	return ast.CreateInlineListNode(&ast.EmbedBLOBNode{
		Blob:   svg,
		Syntax: api.ValueSyntaxSVG,
	})
}

func canvasErrMsg(err error) ast.InlineListNode {
	return ast.CreateInlineListNodeFromWords("Error:", err.Error())
}

func noSVGErrMsg() ast.InlineListNode {
	return ast.CreateInlineListNodeFromWords("NO", "IMAGE")
}
