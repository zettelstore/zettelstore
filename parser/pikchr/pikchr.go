//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package pikchr provides a parser to create SVG from a textual PIC-like description.
package pikchr

import (
	"strconv"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/parser/pikchr/internal"
)

func init() {
	parser.Register(&parser.Info{
		Name:          "pikchr",
		AltNames:      nil,
		IsTextParser:  true,
		IsImageFormat: false,
		ParseBlocks:   parseBlocks,
		ParseInlines:  parseInlines,
	})
}

func parseBlocks(inp *input.Input, m *meta.Meta, _ string) ast.BlockSlice {
	var w, h int
	bsSVG := internal.Pikchr(
		inp.Src[inp.Pos:], "", 0,
		m.GetDefault("width", ""), m.GetDefault("height", ""), getScale(m, "font-scale"),
		&w, &h)
	if w == -1 {
		return ast.BlockSlice{
			&ast.ParaNode{
				Inlines: ast.CreateInlineSliceFromWords("Pikchr", "error:"),
			},
			&ast.VerbatimNode{
				Kind:    ast.VerbatimHTML,
				Content: bsSVG,
			},
		}
	}
	return ast.BlockSlice{&ast.BLOBNode{
		Title:  "",
		Syntax: api.ValueSyntaxSVG,
		Blob:   bsSVG,
	}}
}

func parseInlines(_ *input.Input, syntax string) ast.InlineSlice {
	return ast.CreateInlineSliceFromWords("No", "inline", "code", "allowed", "for", "syntax:", syntax)
}
func getScale(m *meta.Meta, key string) internal.PNum {
	if val, found := m.Get(key); found {
		if scale, err := strconv.ParseFloat(val, 64); err == nil && scale > 0.001 && scale < 1000.0 {
			return internal.PNum(scale)
		}
	}
	return 1.0
}
