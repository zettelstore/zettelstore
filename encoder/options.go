//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
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

import (
	"zettelstore.de/z/ast"
)

// StringOption is an option with a string value
type StringOption struct {
	Key   string
	Value string
}

// Name returns the visible name of this option.
func (so *StringOption) Name() string { return so.Key }

// BoolOption is an option with a boolean value.
type BoolOption struct {
	Key   string
	Value bool
}

// Name returns the visible name of this option.
func (bo *BoolOption) Name() string { return bo.Key }

// TitleOption is an option to give the title as a AST inline slice
type TitleOption struct {
	Inline ast.InlineSlice
}

// Name returns the visible name of this option.
func (mo *TitleOption) Name() string { return "title" }

// StringsOption is an option that have a sequence of strings as the value.
type StringsOption struct {
	Key   string
	Value []string
}

// Name returns the visible name of this option.
func (so *StringsOption) Name() string { return so.Key }

// AdaptLinkOption specifies a link adapter.
type AdaptLinkOption struct {
	Adapter func(*ast.LinkNode) ast.InlineNode
}

// Name returns the visible name of this option.
func (al *AdaptLinkOption) Name() string { return "AdaptLinkOption" }

// AdaptImageOption specifies an image adapter.
type AdaptImageOption struct {
	Adapter func(*ast.ImageNode) ast.InlineNode
}

// Name returns the visible name of this option.
func (al *AdaptImageOption) Name() string { return "AdaptImageOption" }

// AdaptCiteOption specifies a citation adapter.
type AdaptCiteOption struct {
	Adapter func(*ast.CiteNode) ast.InlineNode
}

// Name returns the visible name of this option.
func (al *AdaptCiteOption) Name() string { return "AdaptCiteOption" }
