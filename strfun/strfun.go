//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package strfun provides some string functions.
package strfun

import (
	"strings"
	"unicode/utf8"
)

// Length returns the number of runes in the given string.
func Length(s string) int {
	return utf8.RuneCountInString(s)
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

// SplitLines splits the given string into a list of lines.
func SplitLines(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
}
