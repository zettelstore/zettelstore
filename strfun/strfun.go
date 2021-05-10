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

// JustifyLeft ensures that the string has a defined length.
func JustifyLeft(s string, maxLen int, pad rune) string {
	if maxLen < 1 {
		return ""
	}
	runes := make([]rune, 0, len(s))
	for _, r := range s {
		runes = append(runes, r)
	}
	if len(runes) > maxLen {
		runes = runes[:maxLen]
		runes[maxLen-1] = '\u2025'
	}

	var sb strings.Builder
	for _, r := range runes {
		sb.WriteRune(r)
	}
	for i := 0; i < maxLen-len(runes); i++ {
		sb.WriteRune(pad)
	}
	return sb.String()
}
