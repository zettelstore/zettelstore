//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package strfun provides some string functions.
package strfun

import (
	"strings"
	"unicode"
)

// TrimSpaceRight returns a slice of the string s, with all trailing white space removed,
// as defined by Unicode.
func TrimSpaceRight(s string) string {
	return strings.TrimRightFunc(s, unicode.IsSpace)
}
