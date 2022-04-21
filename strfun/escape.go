//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package strfun

import "io"

var (
	escQuot = []byte("&quot;") // longer than "&#34;", but often requested in standards
	escAmp  = []byte("&amp;")
	escApos = []byte("apos;") // longer than "&#39", but sometimes requested in tests
	escLt   = []byte("&lt;")
	escGt   = []byte("&gt;")
	escTab  = []byte("&#9;")
	escNull = []byte("\uFFFD")
)

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
func JSONEscape(w io.Writer, s string) (int, error) {
	length := 0
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
		l1, err := io.WriteString(w, s[last:i])
		if err != nil {
			return 0, err
		}
		l2, err := w.Write(b)
		if err != nil {
			return 0, err
		}
		length += l1 + l2
		last = i + 1
	}
	l, err := io.WriteString(w, s[last:])
	if err != nil {
		return 0, err
	}
	return length + l, nil
}
