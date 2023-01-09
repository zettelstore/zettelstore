//-----------------------------------------------------------------------------
// Copyright (c) 2020-2023 Detlef Stern
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
	"context"
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
	IsASTParser   bool
	IsTextFormat  bool
	IsImageFormat bool
	ParseBlocks   func(*input.Input, *meta.Meta, string) ast.BlockSlice
	ParseInlines  func(*input.Input, string) ast.InlineSlice
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

// IsASTParser returns whether the given syntax parses text into an AST or not.
func IsASTParser(syntax string) bool {
	pi, ok := registry[syntax]
	if !ok {
		return false
	}
	return pi.IsASTParser
}

// IsTextFormat returns whether the given syntax is known to be a text format.
func IsTextFormat(syntax string) bool {
	pi, ok := registry[syntax]
	if !ok {
		return false
	}
	return pi.IsTextFormat
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
func ParseBlocks(inp *input.Input, m *meta.Meta, syntax string, hi config.HTMLInsecurity) ast.BlockSlice {
	return parseBlocksAndClean(inp, m, syntax, hi)
}
func parseBlocksAndClean(inp *input.Input, m *meta.Meta, syntax string, hi config.HTMLInsecurity) ast.BlockSlice {
	bs := Get(syntax).ParseBlocks(inp, m, syntax)
	cleaner.CleanBlockSlice(&bs, hi.AllowHTML(syntax))
	return bs
}

// ParseInlines parses some input and returns a slice of inline nodes.
func ParseInlines(inp *input.Input, syntax string) ast.InlineSlice {
	// Do not clean, because we don't know the context where this function will be called.
	return Get(syntax).ParseInlines(inp, syntax)
}

// ParseMetadata parses a string as Zettelmarkup, resulting in an inline slice.
// Typically used to parse the title or other metadata of type Zettelmarkup.
func ParseMetadata(value string) ast.InlineSlice {
	return ParseInlines(input.NewInput([]byte(value)), meta.SyntaxZmk)
}

// ParseMetadataNoLink parses a string as Zettelmarkup, resulting in an inline slice.
// All link and footnote nodes will be removed.
func ParseMetadataNoLink(value string) ast.InlineSlice {
	in := ParseMetadata(value)
	cleaner.CleanInlineLinks(&in)
	return in
}

// ParseDescription returns a suitable description stored in the metadata as an inline slice.
func ParseDescription(m *meta.Meta) ast.InlineSlice {
	if m == nil {
		return nil
	}
	descr, found := m.Get(api.KeySummary)
	if !found {
		descr, found = m.Get(api.KeyTitle)
	}
	if !found {
		return nil
	}
	return ParseMetadataNoLink(descr)
}

// ParseZettel parses the zettel based on the syntax.
func ParseZettel(ctx context.Context, zettel domain.Zettel, syntax string, rtConfig config.Config) *ast.ZettelNode {
	m := zettel.Meta
	inhMeta := m
	if rtConfig != nil {
		inhMeta = rtConfig.AddDefaultValues(ctx, inhMeta)
	}
	if syntax == "" {
		syntax, _ = inhMeta.Get(api.KeySyntax)
	}
	parseMeta := inhMeta
	if syntax == meta.SyntaxNone {
		parseMeta = m
	}

	hi := config.NoHTML
	if rtConfig != nil {
		hi = rtConfig.GetHTMLInsecurity()
	}
	return &ast.ZettelNode{
		Meta:    m,
		Content: zettel.Content,
		Zid:     m.Zid,
		InhMeta: inhMeta,
		Ast:     parseBlocksAndClean(input.NewInput(zettel.Content.AsBytes()), parseMeta, syntax, hi),
		Syntax:  syntax,
	}
}
