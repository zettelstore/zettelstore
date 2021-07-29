//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package encoder provides a generic interface to encode the abstract syntax
// tree into some text form.
package encoder

import "zettelstore.de/z/ast"

// Environment specifies all data and functions that affects encoding.
type Environment struct {
	// Important for HTML encoder
	Lang           string // default language
	Interactive    bool   // Encoded data will be placed in interactive content
	Xhtml          bool   // use XHTML syntax instead of HTML syntax
	MarkerExternal string // Marker after link to (external) material.
	NewWindow      bool   // open link in new window
	IgnoreMeta     map[string]bool
	footnotes      []*ast.FootnoteNode // Stores footnotes detected while encoding
}

// IsInteractive returns true, if Interactive is enabled and currently embedded
// interactive encoding will take place.
func (env *Environment) IsInteractive(inInteractive bool) bool {
	return inInteractive && env != nil && env.Interactive
}

// IsXHTML return true, if XHTML is enabled.
func (env *Environment) IsXHTML() bool {
	return env != nil && env.Xhtml
}

// HasNewWindow retruns true, if a new browser windows should be opened.
func (env *Environment) HasNewWindow() bool {
	return env != nil && env.NewWindow
}

// AddFootnote adds a footnote node to the environment and returns the number of that footnote.
func (env *Environment) AddFootnote(fn *ast.FootnoteNode) int {
	if env == nil {
		return 0
	}
	env.footnotes = append(env.footnotes, fn)
	return len(env.footnotes)
}

// GetCleanFootnotes returns the list of remembered footnote and forgets about them.
func (env *Environment) GetCleanFootnotes() []*ast.FootnoteNode {
	if env == nil {
		return nil
	}
	result := env.footnotes
	env.footnotes = nil
	return result
}
