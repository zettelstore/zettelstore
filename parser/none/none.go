//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package none provides a none-parser, e.g. for zettel with just metadata.
package none

import (
	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
)

func init() {
	parser.Register(&parser.Info{
		Name:          api.ValueSyntaxNone,
		AltNames:      []string{},
		IsTextParser:  false,
		IsImageFormat: false,
		ParseBlocks:   parseBlocks,
		ParseInlines:  parseInlines,
	})
}
func parseBlocks(*input.Input, *meta.Meta, string) *ast.BlockListNode {
	return &ast.BlockListNode{}
}

func parseInlines(inp *input.Input, _ string) ast.InlineListNode {
	inp.SkipToEOL()
	return ast.InlineListNode{}
}
