//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package jsonenc encodes the abstract syntax tree into some JSON formats.
package jsonenc

import (
	"bytes"

	"zettelstore.de/z/encoder"
)

var (
	jsBackslash   = []byte{'\\', '\\'}
	jsDoubleQuote = []byte{'\\', '"'}
	jsNewline     = []byte{'\\', 'n'}
	jsTab         = []byte{'\\', 't'}
	jsCr          = []byte{'\\', 'r'}
	jsUnicode     = []byte{'\\', 'u', '0', '0', '0', '0'}
	jsHex         = []byte("0123456789ABCDEF")
)

// Escape returns the given string as a byte slice, where every non-printable
// rune is made printable.
func Escape(s string) []byte {
	var buf bytes.Buffer

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
		buf.WriteString(s[last:i])
		buf.Write(b)
		last = i + 1
	}
	buf.WriteString(s[last:])
	return buf.Bytes()
}

func writeEscaped(b *encoder.BufWriter, s string) {
	b.WriteByte('"')
	b.Write(Escape(s))
	b.WriteByte('"')
}
