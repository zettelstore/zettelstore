//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
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
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	useUnicode = []*unicode.RangeTable{
		unicode.Letter,
		unicode.Number,
	}
	ignoreUnicode = []*unicode.RangeTable{
		unicode.Mark,
		unicode.Sk,
		unicode.Lm,
	}
)

// Slugify returns a string that can be used as part of an URL
func Slugify(s string) string {
	result := make([]rune, 0, len(s))
	addDash := false
	for _, r := range norm.NFKD.String(s) {
		if unicode.IsOneOf(useUnicode, r) {
			result = append(result, unicode.ToLower(r))
			addDash = true
		} else if !unicode.IsOneOf(ignoreUnicode, r) && addDash {
			result = append(result, '-')
			addDash = false
		}
	}
	if i := len(result) - 1; i >= 0 && result[i] == '-' {
		result = result[:i]
	}
	return string(result)
}

// NormalizeWords produces a word list that is normalized for better searching.
func NormalizeWords(s string) []string {
	result := make([]string, 0, 1)
	word := make([]rune, 0, len(s))
	for _, r := range norm.NFKD.String(s) {
		if unicode.IsOneOf(useUnicode, r) {
			word = append(word, unicode.ToLower(r))
		} else if !unicode.IsOneOf(ignoreUnicode, r) && len(word) > 0 {
			result = append(result, string(word))
			word = word[:0]
		}
	}
	if len(word) > 0 {
		result = append(result, string(word))
	}
	return result
}
