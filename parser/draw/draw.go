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
	defaultFont   = ""
	defaultScaleX = 10
	defaultScaleY = 20
)

func parseBlocks(inp *input.Input, m *meta.Meta, _ string) *ast.BlockListNode {
	font := m.GetDefault("font", defaultFont)
	scaleX := m.GetNumber("x-scale", defaultScaleX)
	scaleY := m.GetNumber("y-scale", defaultScaleY)
	iln := parseDraw(inp, font, scaleX, scaleY)
	if iln == nil {
		return nil
	}
	return &ast.BlockListNode{List: []ast.BlockNode{&ast.ParaNode{Inlines: iln}}}
}

func parseInlines(inp *input.Input, _ string) *ast.InlineListNode {
	return parseDraw(inp, defaultFont, defaultScaleX, defaultScaleY)
}

func parseDraw(inp *input.Input, font string, scaleX, scaleY int) *ast.InlineListNode {
	canvas, err := newCanvas(inp.Src[inp.Pos:], 8)
	if err != nil {
		return &ast.InlineListNode{
			List: []ast.InlineNode{
				&ast.TextNode{Text: "Error:"},
				&ast.SpaceNode{Lexeme: " "},
				&ast.TextNode{Text: err.Error()},
			},
		}
	}
	svg := canvasToSVG(canvas, font, scaleX, scaleY)
	if len(svg) == 0 {
		return &ast.InlineListNode{
			List: []ast.InlineNode{
				&ast.TextNode{Text: "NO"},
				&ast.SpaceNode{Lexeme: " "},
				&ast.TextNode{Text: "IMAGE"},
			},
		}
	}

	return ast.CreateInlineListNode(&ast.EmbedBLOBNode{
		Blob:   svg,
		Syntax: api.ValueSyntaxSVG,
	})
}
