//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
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

	"zettelstore.de/c/api"
	"zettelstore.de/c/zjson"
	"zettelstore.de/z/ast"
)

const (
	defaultTabSize = 8
	defaultFont    = ""
	defaultScaleX  = 10
	defaultScaleY  = 20
)

// ParseDrawBlock parses the content of an eval verbatim node into an SVG image BLOB.
func ParseDrawBlock(vn *ast.VerbatimNode) ast.BlockNode {
	font := defaultFont
	if val, found := vn.Attrs.Get("font"); found {
		font = val
	}
	scaleX := getScale(vn.Attrs, "x-scale", defaultScaleX)
	scaleY := getScale(vn.Attrs, "y-scale", defaultScaleY)

	canvas, err := newCanvas(vn.Content, defaultTabSize)
	if err != nil {
		return ast.CreateParaNode(canvasErrMsg(err)...)
	}
	if scaleX < 1 || 1000000 < scaleX {
		scaleX = defaultScaleX
	}
	if scaleY < 1 || 1000000 < scaleY {
		scaleY = defaultScaleY
	}
	svg := canvasToSVG(canvas, font, int(scaleX), int(scaleY))
	if len(svg) == 0 {
		return ast.CreateParaNode(noSVGErrMsg()...)
	}
	return &ast.BLOBNode{
		Title:  "",
		Syntax: api.ValueSyntaxSVG,
		Blob:   svg,
	}
}

func getScale(attrs zjson.Attributes, key string, defVal int) int {
	if val, found := attrs.Get(key); found {
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
