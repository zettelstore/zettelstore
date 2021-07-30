//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package parser provides a generic interface to a range of different parsers.
package parser

import (
	"log"

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
	Name         string
	AltNames     []string
	ParseBlocks  func(*input.Input, *meta.Meta, string) *ast.BlockListNode
	ParseInlines func(*input.Input, string) *ast.InlineListNode
}

var registry = map[string]*Info{}

// Register the parser (info) for later retrieval.
func Register(pi *Info) *Info {
	if _, ok := registry[pi.Name]; ok {
		log.Fatalf("Parser %q already registered", pi.Name)
	}
	registry[pi.Name] = pi
	for _, alt := range pi.AltNames {
		if _, ok := registry[alt]; ok {
			log.Fatalf("Parser %q already registered", alt)
		}
		registry[alt] = pi
	}
	return pi
}

// Get the parser (info) by name. If name not found, use a default parser.
func Get(name string) *Info {
	if pi := registry[name]; pi != nil {
		return pi
	}
	if pi := registry["plain"]; pi != nil {
		return pi
	}
	log.Printf("No parser for %q found", name)
	panic("No default parser registered")
}

// ParseBlocks parses some input and returns a slice of block nodes.
func ParseBlocks(inp *input.Input, m *meta.Meta, syntax string) *ast.BlockListNode {
	bs := Get(syntax).ParseBlocks(inp, m, syntax)
	cleaner.CleanupBlockSlice(bs.List)
	return bs
}

// ParseInlines parses some input and returns a slice of inline nodes.
func ParseInlines(inp *input.Input, syntax string) *ast.InlineListNode {
	return Get(syntax).ParseInlines(inp, syntax)
}

// ParseMetadata parses a string as Zettelmarkup, resulting in an inline slice.
// Typically used to parse the title or other metadata of type Zettelmarkup.
func ParseMetadata(title string) *ast.InlineListNode {
	return ParseInlines(input.NewInput(title), meta.ValueSyntaxZmk)
}

// ParseZettel parses the zettel based on the syntax.
func ParseZettel(zettel domain.Zettel, syntax string, rtConfig config.Config) *ast.ZettelNode {
	m := zettel.Meta
	inhMeta := rtConfig.AddDefaultValues(m)
	if syntax == "" {
		syntax, _ = inhMeta.Get(meta.KeySyntax)
	}
	parseMeta := inhMeta
	if syntax == meta.ValueSyntaxNone {
		parseMeta = m
	}
	return &ast.ZettelNode{
		Meta:    m,
		Content: zettel.Content,
		Zid:     m.Zid,
		InhMeta: inhMeta,
		Ast:     ParseBlocks(input.NewInput(zettel.Content.AsString()), parseMeta, syntax),
	}
}
