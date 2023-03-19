//-----------------------------------------------------------------------------
// Copyright (c) 2022-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package draw provides a parser to create SVG from ASCII drawing.
//
// It is not a parser registered by the general parser framework (directed by
// metadata "syntax" of a zettel). It will be used when a zettel is evaluated.
package draw

import (
	"strconv"

	"zettelstore.de/c/attrs"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
)

func init() {
	parser.Register(&parser.Info{
		Name:          meta.SyntaxDraw,
		AltNames:      []string{},
		IsASTParser:   true,
		IsTextFormat:  true,
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

func parseBlocks(inp *input.Input, m *meta.Meta, _ string) ast.BlockSlice {
	font := m.GetDefault("font", defaultFont)
	scaleX := m.GetNumber("x-scale", defaultScaleX)
	scaleY := m.GetNumber("y-scale", defaultScaleY)
	if scaleX < 1 || 1000000 < scaleX {
		scaleX = defaultScaleX
	}
	if scaleY < 1 || 1000000 < scaleY {
		scaleY = defaultScaleY
	}

	canvas, err := newCanvas(inp.Src[inp.Pos:])
	if err != nil {
		return ast.BlockSlice{ast.CreateParaNode(canvasErrMsg(err)...)}
	}
	svg := canvasToSVG(canvas, font, int(scaleX), int(scaleY))
	if len(svg) == 0 {
		return ast.BlockSlice{ast.CreateParaNode(noSVGErrMsg()...)}
	}
	return ast.BlockSlice{&ast.BLOBNode{
		Description: parser.ParseDescription(m),
		Syntax:      meta.SyntaxSVG,
		Blob:        svg,
	}}
}

func parseInlines(inp *input.Input, _ string) ast.InlineSlice {
	canvas, err := newCanvas(inp.Src[inp.Pos:])
	if err != nil {
		return canvasErrMsg(err)
	}
	svg := canvasToSVG(canvas, defaultFont, defaultScaleX, defaultScaleY)
	if len(svg) == 0 {
		return noSVGErrMsg()
	}
	return ast.InlineSlice{&ast.EmbedBLOBNode{
		Attrs:   nil,
		Syntax:  meta.SyntaxSVG,
		Blob:    svg,
		Inlines: nil,
	}}
}

// ParseDrawBlock parses the content of an eval verbatim node into an SVG image BLOB.
func ParseDrawBlock(vn *ast.VerbatimNode) ast.BlockNode {
	font := defaultFont
	if val, found := vn.Attrs.Get("font"); found {
		font = val
	}
	scaleX := getScale(vn.Attrs, "x-scale", defaultScaleX)
	scaleY := getScale(vn.Attrs, "y-scale", defaultScaleY)

	canvas, err := newCanvas(vn.Content)
	if err != nil {
		return ast.CreateParaNode(canvasErrMsg(err)...)
	}
	if scaleX < 1 || 1000000 < scaleX {
		scaleX = defaultScaleX
	}
	if scaleY < 1 || 1000000 < scaleY {
		scaleY = defaultScaleY
	}
	svg := canvasToSVG(canvas, font, scaleX, scaleY)
	if len(svg) == 0 {
		return ast.CreateParaNode(noSVGErrMsg()...)
	}
	return &ast.BLOBNode{
		Description: nil, // TODO: look for attribute "summary" / "title"
		Syntax:      meta.SyntaxSVG,
		Blob:        svg,
	}
}

func getScale(a attrs.Attributes, key string, defVal int) int {
	if val, found := a.Get(key); found {
		if n, err := strconv.Atoi(val); err == nil && 0 < n && n < 100000 {
			return n
		}
	}
	return defVal
}

func canvasErrMsg(err error) ast.InlineSlice {
	return ast.CreateInlineSliceFromWords("Error:", err.Error())
}

func noSVGErrMsg() ast.InlineSlice {
	return ast.CreateInlineSliceFromWords("NO", "IMAGE")
}
