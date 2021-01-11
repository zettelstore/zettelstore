//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package plain provides a parser for plain text data.
package plain

import (
	"strings"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/runes"
)

func init() {
	parser.Register(&parser.Info{
		Name:         "txt",
		AltNames:     []string{"plain", "text"},
		ParseBlocks:  parseBlocks,
		ParseInlines: parseInlines,
	})
	parser.Register(&parser.Info{
		Name:         "css",
		ParseBlocks:  parseBlocks,
		ParseInlines: parseInlines,
	})
	parser.Register(&parser.Info{
		Name:         "svg",
		ParseBlocks:  parseSVGBlocks,
		ParseInlines: parseSVGInlines,
	})
	parser.Register(&parser.Info{
		Name:         "mustache",
		ParseBlocks:  parseBlocks,
		ParseInlines: parseInlines,
	})
}

func parseBlocks(inp *input.Input, m *meta.Meta, syntax string) ast.BlockSlice {
	return ast.BlockSlice{
		&ast.VerbatimNode{
			Code:  ast.VerbatimProg,
			Attrs: &ast.Attributes{Attrs: map[string]string{"": syntax}},
			Lines: readLines(inp),
		},
	}
}

func readLines(inp *input.Input) (lines []string) {
	for {
		inp.EatEOL()
		posL := inp.Pos
		switch inp.Ch {
		case input.EOS:
			return lines
		}
		inp.SkipToEOL()
		lines = append(lines, inp.Src[posL:inp.Pos])
	}
}

func parseInlines(inp *input.Input, syntax string) ast.InlineSlice {
	inp.SkipToEOL()
	return ast.InlineSlice{
		&ast.LiteralNode{
			Code:  ast.LiteralProg,
			Attrs: &ast.Attributes{Attrs: map[string]string{"": syntax}},
			Text:  inp.Src[0:inp.Pos],
		},
	}
}

func parseSVGBlocks(inp *input.Input, m *meta.Meta, syntax string) ast.BlockSlice {
	ins := parseSVGInlines(inp, syntax)
	if ins == nil {
		return nil
	}
	return ast.BlockSlice{
		&ast.ParaNode{
			Inlines: ins,
		},
	}
}

func parseSVGInlines(inp *input.Input, syntax string) ast.InlineSlice {
	svgSrc := scanSVG(inp)
	if svgSrc == "" {
		return nil
	}
	return ast.InlineSlice{
		&ast.ImageNode{
			Blob:   []byte(svgSrc),
			Syntax: syntax,
		},
	}
}

func scanSVG(inp *input.Input) string {
	for runes.IsSpace(inp.Ch) {
		inp.Next()
	}
	svgSrc := inp.Src[inp.Pos:]
	if !strings.HasPrefix(svgSrc, "<svg ") {
		return ""
	}
	// TODO: check proper end </svg>
	return svgSrc
}
