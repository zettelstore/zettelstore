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
	// Important for many encoder.
	LinkAdapter  func(*ast.LinkNode) ast.InlineNode
	ImageAdapter func(*ast.ImageNode) ast.InlineNode
	CiteAdapter  func(*ast.CiteNode) ast.InlineNode

	// Important for HTML encoder
	Lang           string // default language
	Interactive    bool   // Encoded data will be placed in interactive content
	Xhtml          bool   // use XHTML syntax instead of HTML syntax
	MarkerExternal string // Marker after link to (external) material.
	NewWindow      bool   // open link in new window
	IgnoreMeta     map[string]bool
}

// AdaptLink helps to call the link adapter.
func (env *Environment) AdaptLink(ln *ast.LinkNode) (*ast.LinkNode, ast.InlineNode) {
	if env == nil || env.LinkAdapter == nil {
		return ln, nil
	}
	n := env.LinkAdapter(ln)
	if n == nil {
		return ln, nil
	}
	if ln2, ok := n.(*ast.LinkNode); ok {
		return ln2, nil
	}
	return nil, n
}

// AdaptImage helps to call the link adapter.
func (env *Environment) AdaptImage(in *ast.ImageNode) (*ast.ImageNode, ast.InlineNode) {
	if env == nil || env.ImageAdapter == nil {
		return in, nil
	}
	n := env.ImageAdapter(in)
	if n == nil {
		return in, nil
	}
	if in2, ok := n.(*ast.ImageNode); ok {
		return in2, nil
	}
	return nil, n
}

// AdaptCite helps to call the link adapter.
func (env *Environment) AdaptCite(cn *ast.CiteNode) (*ast.CiteNode, ast.InlineNode) {
	if env == nil || env.CiteAdapter == nil {
		return cn, nil
	}
	n := env.CiteAdapter(cn)
	if n == nil {
		return cn, nil
	}
	if cn2, ok := n.(*ast.CiteNode); ok {
		return cn2, nil
	}
	return nil, n
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
