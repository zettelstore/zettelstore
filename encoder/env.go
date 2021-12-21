//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

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
	footnotes      []footnoteInfo // Stores footnotes detected while encoding
	footnoteNum    int
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

type footnoteInfo struct {
	fn  *ast.FootnoteNode
	num int
}

// AddFootnote adds a footnote node to the environment and returns the number of that footnote.
func (env *Environment) AddFootnote(fn *ast.FootnoteNode) int {
	if env == nil {
		return 0
	}
	env.footnoteNum++
	env.footnotes = append(env.footnotes, footnoteInfo{fn: fn, num: env.footnoteNum})
	return env.footnoteNum
}

// PopFootnote returns the next footnote and removes it from the list.
func (env *Environment) PopFootnote() (*ast.FootnoteNode, int) {
	if env == nil {
		return nil, -1
	}
	if len(env.footnotes) == 0 {
		env.footnotes = nil
		env.footnoteNum = 0
		return nil, -1
	}
	fni := env.footnotes[0]
	env.footnotes = env.footnotes[1:]
	return fni.fn, fni.num
}
