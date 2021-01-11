//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package domain provides domain specific types, constants, and functions.
package domain

import (
	"unicode/utf8"
)

// Content is just the uninterpreted content of a zettel.
type Content string

// NewContent creates a new content from a string.
func NewContent(s string) Content { return Content(s) }

// AsString returns the content itself is a string.
func (zc Content) AsString() string { return string(zc) }

// AsBytes returns the content itself is a byte slice.
func (zc Content) AsBytes() []byte { return []byte(zc) }

// IsBinary returns true if the content contains non-unicode values or is,
// interpreted a text, with a high probability binary content.
func (zc Content) IsBinary() bool {
	s := string(zc)
	if !utf8.ValidString(s) {
		return true
	}
	l := len(s)
	for i := 0; i < l; i++ {
		if s[i] == 0 {
			return true
		}
	}
	return false
}
