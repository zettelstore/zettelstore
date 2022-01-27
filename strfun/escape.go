//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
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
	"io"
	"strings"
)

const (
	htmlQuot     = "&quot;" // longer than "&#34;", but often requested in standards
	htmlAmp      = "&amp;"
	htmlLt       = "&lt;"
	htmlGt       = "&gt;"
	htmlNull     = "\uFFFD"
	htmlVisSpace = "\u2423"
)

var (
	htmlEscapes = []string{`&`, htmlAmp,
		`<`, htmlLt,
		`>`, htmlGt,
		`"`, htmlQuot,
		"\000", htmlNull,
	}
	htmlEscaper    = strings.NewReplacer(htmlEscapes...)
	htmlVisEscapes = append(htmlEscapes,
		" ", htmlVisSpace,
		"\u00a0", htmlVisSpace,
	)
	htmlVisEscaper = strings.NewReplacer(htmlVisEscapes...)
)

// HTMLEscape writes to w the escaped HTML equivalent of the given string.
func HTMLEscape(w io.Writer, s string) (int, error) { return htmlEscaper.WriteString(w, s) }

// HTMLEscapeVisible writes to w the escaped HTML equivalent of the given string.
// Each space is written as U-2423.
func HTMLEscapeVisible(w io.Writer, s string) (int, error) { return htmlVisEscaper.WriteString(w, s) }

var (
	escQuot = []byte(htmlQuot) // longer than "&#34;", but often requested in standards
	escAmp  = []byte(htmlAmp)
	escApos = []byte("apos;") // longer than "&#39", but sometimes requested in tests
	escLt   = []byte(htmlLt)
	escGt   = []byte(htmlGt)
	escTab  = []byte("&#9;")
	escNull = []byte(htmlNull)
)

// HTMLAttrEscape writes to w the escaped HTML equivalent of the given string to be used
// in attributes.
func HTMLAttrEscape(w io.Writer, s string) {
	last := 0
	var html []byte
	lenS := len(s)
	for i := 0; i < lenS; i++ {
		switch s[i] {
		case '\000':
			html = escNull
		case '"':
			html = escQuot
		case '&':
			html = escAmp
		default:
			continue
		}
		io.WriteString(w, s[last:i])
		w.Write(html)
		last = i + 1
	}
	io.WriteString(w, s[last:])
}

// XMLEscape writes the string to the given writer, where every rune that has a special
// meaning in XML is escaped.
func XMLEscape(w io.Writer, s string) {
	var esc []byte
	last := 0
	for i, ch := range s {
		switch ch {
		case '\000':
			esc = escNull
		case '"':
			esc = escQuot
		case '\'':
			esc = escApos
		case '&':
			esc = escAmp
		case '<':
			esc = escLt
		case '>':
			esc = escGt
		case '\t':
			esc = escTab
		default:
			continue
		}
		io.WriteString(w, s[last:i])
		w.Write(esc)
		last = i + 1
	}
	io.WriteString(w, s[last:])
}

var (
	jsBackslash   = []byte{'\\', '\\'}
	jsDoubleQuote = []byte{'\\', '"'}
	jsNewline     = []byte{'\\', 'n'}
	jsTab         = []byte{'\\', 't'}
	jsCr          = []byte{'\\', 'r'}
	jsUnicode     = []byte{'\\', 'u', '0', '0', '0', '0'}
	jsHex         = []byte("0123456789ABCDEF")
)

// JSONEscape returns the given string as a byte slice, where every non-printable
// rune is made printable.
func JSONEscape(w io.Writer, s string) {
	last := 0
	for i, ch := range s {
		var b []byte
		switch ch {
		case '\t':
			b = jsTab
		case '\r':
			b = jsCr
		case '\n':
			b = jsNewline
		case '"':
			b = jsDoubleQuote
		case '\\':
			b = jsBackslash
		default:
			if ch < ' ' {
				b = jsUnicode
				b[2] = '0'
				b[3] = '0'
				b[4] = jsHex[ch>>4]
				b[5] = jsHex[ch&0xF]
			} else {
				continue
			}
		}
		io.WriteString(w, s[last:i])
		w.Write(b)
		last = i + 1
	}
	io.WriteString(w, s[last:])
}
