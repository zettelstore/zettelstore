//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package plain provides a parser for plain text data.
package plain

import (
	"strings"

	"zettelstore.de/c/attrs"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
)

func init() {
	parser.Register(&parser.Info{
		Name:          meta.SyntaxTxt,
		AltNames:      []string{meta.SyntaxPlain, meta.SyntaxText},
		IsASTParser:   false,
		IsTextFormat:  true,
		IsImageFormat: false,
		ParseBlocks:   parseBlocks,
		ParseInlines:  parseInlines,
	})
	parser.Register(&parser.Info{
		Name:          meta.SyntaxHTML,
		AltNames:      []string{},
		IsASTParser:   false,
		IsTextFormat:  true,
		IsImageFormat: false,
		ParseBlocks:   parseBlocksHTML,
		ParseInlines:  parseInlinesHTML,
	})
	parser.Register(&parser.Info{
		Name:          meta.SyntaxCSS,
		AltNames:      []string{},
		IsASTParser:   false,
		IsTextFormat:  true,
		IsImageFormat: false,
		ParseBlocks:   parseBlocks,
		ParseInlines:  parseInlines,
	})
	parser.Register(&parser.Info{
		Name:          meta.SyntaxSVG,
		AltNames:      []string{},
		IsASTParser:   false,
		IsTextFormat:  true,
		IsImageFormat: true,
		ParseBlocks:   parseSVGBlocks,
		ParseInlines:  parseSVGInlines,
	})
	parser.Register(&parser.Info{
		Name:          meta.SyntaxMustache,
		AltNames:      []string{},
		IsASTParser:   false,
		IsTextFormat:  true,
		IsImageFormat: false,
		ParseBlocks:   parseBlocks,
		ParseInlines:  parseInlines,
	})
}

func parseBlocks(inp *input.Input, _ *meta.Meta, syntax string) ast.BlockSlice {
	return doParseBlocks(inp, syntax, ast.VerbatimProg)
}
func parseBlocksHTML(inp *input.Input, _ *meta.Meta, syntax string) ast.BlockSlice {
	return doParseBlocks(inp, syntax, ast.VerbatimHTML)
}
func doParseBlocks(inp *input.Input, syntax string, kind ast.VerbatimKind) ast.BlockSlice {
	return ast.BlockSlice{
		&ast.VerbatimNode{
			Kind:    kind,
			Attrs:   attrs.Attributes{"": syntax},
			Content: inp.ScanLineContent(),
		},
	}
}

func parseInlines(inp *input.Input, syntax string) ast.InlineSlice {
	return doParseInlines(inp, syntax, ast.LiteralProg)
}
func parseInlinesHTML(inp *input.Input, syntax string) ast.InlineSlice {
	return doParseInlines(inp, syntax, ast.LiteralHTML)
}
func doParseInlines(inp *input.Input, syntax string, kind ast.LiteralKind) ast.InlineSlice {
	inp.SkipToEOL()
	return ast.InlineSlice{&ast.LiteralNode{
		Kind:    kind,
		Attrs:   attrs.Attributes{"": syntax},
		Content: append([]byte(nil), inp.Src[0:inp.Pos]...),
	}}
}

func parseSVGBlocks(inp *input.Input, _ *meta.Meta, syntax string) ast.BlockSlice {
	is := parseSVGInlines(inp, syntax)
	if len(is) == 0 {
		return nil
	}
	return ast.BlockSlice{ast.CreateParaNode(is...)}
}

func parseSVGInlines(inp *input.Input, syntax string) ast.InlineSlice {
	svgSrc := scanSVG(inp)
	if svgSrc == "" {
		return nil
	}
	return ast.InlineSlice{&ast.EmbedBLOBNode{
		Blob:   []byte(svgSrc),
		Syntax: syntax,
	}}
}

func scanSVG(inp *input.Input) string {
	for input.IsSpace(inp.Ch) {
		inp.Next()
	}
	svgSrc := string(inp.Src[inp.Pos:])
	if !strings.HasPrefix(svgSrc, "<svg ") {
		return ""
	}
	// TODO: check proper end </svg>
	return svgSrc
}
