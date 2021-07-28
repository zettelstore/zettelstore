//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package strfun provides some string functions.
package strfun

import "io"

var (
	htmlQuot     = []byte("&quot;") // shorter than "&39;", but often requested in standards
	htmlAmp      = []byte("&amp;")
	htmlLt       = []byte("&lt;")
	htmlGt       = []byte("&gt;")
	htmlNull     = []byte("\uFFFD")
	htmlVisSpace = []byte("\u2423")
)

// HTMLEscape writes to w the escaped HTML equivalent of the given string.
// If visibleSpace is true, each space is written as U-2423.
func HTMLEscape(w io.Writer, s string, visibleSpace bool) {
	last := 0
	var html []byte
	lenS := len(s)
	for i := 0; i < lenS; i++ {
		switch s[i] {
		case '\000':
			html = htmlNull
		case ' ':
			if visibleSpace {
				html = htmlVisSpace
			} else {
				continue
			}
		case '"':
			html = htmlQuot
		case '&':
			html = htmlAmp
		case '<':
			html = htmlLt
		case '>':
			html = htmlGt
		default:
			continue
		}
		io.WriteString(w, s[last:i])
		w.Write(html)
		last = i + 1
	}
	io.WriteString(w, s[last:])
}

// HTMLAttrEscape writes to w the escaped HTML equivalent of the given string to be used
// in attributes.
func HTMLAttrEscape(w io.Writer, s string) {
	last := 0
	var html []byte
	lenS := len(s)
	for i := 0; i < lenS; i++ {
		switch s[i] {
		case '\000':
			html = htmlNull
		case '"':
			html = htmlQuot
		case '&':
			html = htmlAmp
		default:
			continue
		}
		io.WriteString(w, s[last:i])
		w.Write(html)
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
