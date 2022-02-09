//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package parser provides a generic interface to a range of different parsers.
package parser

import (
	"fmt"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser/cleaner"
)

// Info describes a single parser.
//
// Before ParseBlocks() or ParseInlines() is called, ensure the input stream to
// be valid. This can ce achieved on calling inp.Next() after the input stream
// was created.
type Info struct {
	Name          string
	AltNames      []string
	IsTextParser  bool
	IsImageFormat bool
	ParseBlocks   func(*input.Input, *meta.Meta, string) *ast.BlockListNode
	ParseInlines  func(*input.Input, string) ast.InlineListNode
}

var registry = map[string]*Info{}

// Register the parser (info) for later retrieval.
func Register(pi *Info) {
	if _, ok := registry[pi.Name]; ok {
		panic(fmt.Sprintf("Parser %q already registered", pi.Name))
	}
	registry[pi.Name] = pi
	for _, alt := range pi.AltNames {
		if _, ok := registry[alt]; ok {
			panic(fmt.Sprintf("Parser %q already registered", alt))
		}
		registry[alt] = pi
	}
}

// GetSyntaxes returns a list of syntaxes implemented by all registered parsers.
func GetSyntaxes() []string {
	result := make([]string, 0, len(registry))
	for syntax := range registry {
		result = append(result, syntax)
	}
	return result
}

// Get the parser (info) by name. If name not found, use a default parser.
func Get(name string) *Info {
	if pi := registry[name]; pi != nil {
		return pi
	}
	if pi := registry["plain"]; pi != nil {
		return pi
	}
	panic(fmt.Sprintf("No parser for %q found", name))
}

// IsTextParser returns whether the given syntax parses text into an AST or not.
func IsTextParser(syntax string) bool {
	pi, ok := registry[syntax]
	if !ok {
		return false
	}
	return pi.IsTextParser
}

// IsImageFormat returns whether the given syntax is known to be an image format.
func IsImageFormat(syntax string) bool {
	pi, ok := registry[syntax]
	if !ok {
		return false
	}
	return pi.IsImageFormat
}

// ParseBlocks parses some input and returns a slice of block nodes.
func ParseBlocks(inp *input.Input, m *meta.Meta, syntax string) *ast.BlockListNode {
	bln := Get(syntax).ParseBlocks(inp, m, syntax)
	cleaner.CleanBlockList(bln)
	return bln
}

// ParseInlines parses some input and returns a slice of inline nodes.
func ParseInlines(inp *input.Input, syntax string) ast.InlineListNode {
	return Get(syntax).ParseInlines(inp, syntax)
}

// ParseMetadata parses a string as Zettelmarkup, resulting in an inline slice.
// Typically used to parse the title or other metadata of type Zettelmarkup.
func ParseMetadata(value string) ast.InlineListNode {
	return ParseInlines(input.NewInput([]byte(value)), api.ValueSyntaxZmk)
}

// ParseZettel parses the zettel based on the syntax.
func ParseZettel(zettel domain.Zettel, syntax string, rtConfig config.Config) *ast.ZettelNode {
	m := zettel.Meta
	inhMeta := m
	if rtConfig != nil {
		inhMeta = rtConfig.AddDefaultValues(inhMeta)
	}
	if syntax == "" {
		syntax, _ = inhMeta.Get(api.KeySyntax)
	}
	parseMeta := inhMeta
	if syntax == api.ValueSyntaxNone {
		parseMeta = m
	}
	return &ast.ZettelNode{
		Meta:    m,
		Content: zettel.Content,
		Zid:     m.Zid,
		InhMeta: inhMeta,
		Ast:     ParseBlocks(input.NewInput(zettel.Content.AsBytes()), parseMeta, syntax),
		Syntax:  syntax,
	}
}
