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

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
)

func init() {
	parser.Register(&parser.Info{
		Name:          "txt",
		AltNames:      []string{"plain", api.ValueSyntaxText},
		IsTextParser:  false,
		IsImageFormat: false,
		ParseBlocks:   parseBlocks,
		ParseInlines:  parseInlines,
	})
	parser.Register(&parser.Info{
		Name:          api.ValueSyntaxHTML,
		AltNames:      []string{},
		IsTextParser:  false,
		IsImageFormat: false,
		ParseBlocks:   parseBlocksHTML,
		ParseInlines:  parseInlinesHTML,
	})
	parser.Register(&parser.Info{
		Name:          "css",
		AltNames:      []string{},
		IsTextParser:  false,
		IsImageFormat: false,
		ParseBlocks:   parseBlocks,
		ParseInlines:  parseInlines,
	})
	parser.Register(&parser.Info{
		Name:          api.ValueSyntaxSVG,
		AltNames:      []string{},
		IsTextParser:  false,
		IsImageFormat: true,
		ParseBlocks:   parseSVGBlocks,
		ParseInlines:  parseSVGInlines,
	})
	parser.Register(&parser.Info{
		Name:          "mustache",
		AltNames:      []string{},
		IsTextParser:  false,
		IsImageFormat: false,
		ParseBlocks:   parseBlocks,
		ParseInlines:  parseInlines,
	})
}

func parseBlocks(inp *input.Input, _ *meta.Meta, syntax string) *ast.BlockListNode {
	return doParseBlocks(inp, syntax, ast.VerbatimProg)
}
func parseBlocksHTML(inp *input.Input, _ *meta.Meta, syntax string) *ast.BlockListNode {
	return doParseBlocks(inp, syntax, ast.VerbatimHTML)
}
func doParseBlocks(inp *input.Input, syntax string, kind ast.VerbatimKind) *ast.BlockListNode {
	return &ast.BlockListNode{List: []ast.BlockNode{
		&ast.VerbatimNode{
			Kind:    kind,
			Attrs:   &ast.Attributes{Attrs: map[string]string{"": syntax}},
			Content: readContent(inp),
		},
	}}
}

func readContent(inp *input.Input) []byte {
	result := make([]byte, 0, len(inp.Src)-inp.Pos+1)
	for {
		inp.EatEOL()
		posL := inp.Pos
		if inp.Ch == input.EOS {
			return result
		}
		inp.SkipToEOL()
		if len(result) > 0 {
			result = append(result, '\n')
		}
		result = append(result, inp.Src[posL:inp.Pos]...)
	}
}

func parseInlines(inp *input.Input, syntax string) *ast.InlineListNode {
	return doParseInlines(inp, syntax, ast.LiteralProg)
}
func parseInlinesHTML(inp *input.Input, syntax string) *ast.InlineListNode {
	return doParseInlines(inp, syntax, ast.LiteralHTML)
}
func doParseInlines(inp *input.Input, syntax string, kind ast.LiteralKind) *ast.InlineListNode {
	inp.SkipToEOL()
	return ast.CreateInlineListNode(&ast.LiteralNode{
		Kind:    kind,
		Attrs:   &ast.Attributes{Attrs: map[string]string{"": syntax}},
		Content: append([]byte(nil), inp.Src[0:inp.Pos]...),
	})
}

func parseSVGBlocks(inp *input.Input, _ *meta.Meta, syntax string) *ast.BlockListNode {
	iln := parseSVGInlines(inp, syntax)
	if iln == nil {
		return nil
	}
	return &ast.BlockListNode{List: []ast.BlockNode{&ast.ParaNode{Inlines: iln}}}
}

func parseSVGInlines(inp *input.Input, syntax string) *ast.InlineListNode {
	svgSrc := scanSVG(inp)
	if svgSrc == "" {
		return nil
	}
	return ast.CreateInlineListNode(&ast.EmbedBLOBNode{
		Blob:   []byte(svgSrc),
		Syntax: syntax,
	})
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
