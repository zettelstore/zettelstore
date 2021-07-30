//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package encfun provides some helper function to work with encodings.
package encfun

import (
	"strings"

	"zettelstore.de/z/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/parser"
)

// MetaAsInlineList returns the value of the given metadata key as an inlince list.
func MetaAsInlineList(m *meta.Meta, key string) *ast.InlineListNode {
	return parser.ParseMetadata(m.GetDefault(key, ""))
}

// MetaAsText returns the value of given metadata as text.
func MetaAsText(m *meta.Meta, key string) string {
	textEncoder := encoder.Create(api.EncoderText, nil)
	var sb strings.Builder
	_, err := textEncoder.WriteInlines(&sb, MetaAsInlineList(m, key).List)
	if err == nil {
		return sb.String()
	}
	return ""
}
