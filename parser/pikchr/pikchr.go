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
	"github.com/gopikchr/gopikchr"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
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

func parseBlocks(inp *input.Input, _ *meta.Meta, _ string) ast.BlockSlice {
	sSVG, _, _, err := gopikchr.Convert(string(inp.Src[inp.Pos:]))
	if err != nil {
		return ast.BlockSlice{&ast.ParaNode{
			Inlines: ast.CreateInlineSliceFromWords("Error:", err.Error()),
		}}
	}
	return ast.BlockSlice{&ast.BLOBNode{
		Title:  "",
		Syntax: api.ValueSyntaxSVG,
		Blob:   []byte(sSVG),
	}}
}

func parseInlines(_ *input.Input, syntax string) ast.InlineSlice {
	return ast.CreateInlineSliceFromWords("No", "inline", "code", "allowed", "for", "syntax:", syntax)
}
