//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package blob provides a parser of binary data.
package blob

import (
	"zettelstore.de/z/ast"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/zettel/meta"
)

func init() {
	parser.Register(&parser.Info{
		Name:          meta.SyntaxGif,
		AltNames:      nil,
		IsASTParser:   false,
		IsTextFormat:  false,
		IsImageFormat: true,
		ParseBlocks:   parseBlocks,
		ParseInlines:  parseInlines,
	})
	parser.Register(&parser.Info{
		Name:          meta.SyntaxJPEG,
		AltNames:      []string{meta.SyntaxJPG},
		IsASTParser:   false,
		IsTextFormat:  false,
		IsImageFormat: true,
		ParseBlocks:   parseBlocks,
		ParseInlines:  parseInlines,
	})
	parser.Register(&parser.Info{
		Name:          meta.SyntaxPNG,
		AltNames:      nil,
		IsASTParser:   false,
		IsTextFormat:  false,
		IsImageFormat: true,
		ParseBlocks:   parseBlocks,
		ParseInlines:  parseInlines,
	})
	parser.Register(&parser.Info{
		Name:          meta.SyntaxWebp,
		AltNames:      nil,
		IsASTParser:   false,
		IsTextFormat:  false,
		IsImageFormat: true,
		ParseBlocks:   parseBlocks,
		ParseInlines:  parseInlines,
	})
}

func parseBlocks(inp *input.Input, m *meta.Meta, syntax string) ast.BlockSlice {
	if p := parser.Get(syntax); p != nil {
		syntax = p.Name
	}
	return ast.BlockSlice{&ast.BLOBNode{
		Description: parser.ParseDescription(m),
		Syntax:      syntax,
		Blob:        []byte(inp.Src),
	}}
}

func parseInlines(*input.Input, string) ast.InlineSlice { return nil }
