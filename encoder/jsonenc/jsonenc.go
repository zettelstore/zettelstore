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
	"io"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
)

func init() {
	encoder.Register("json", encoder.Info{
		Create:  func() encoder.Encoder { return &jsonEncoder{} },
		Default: true,
	})
}

// jsonEncoder is just a stub. It is not implemented. The real implementation
// is in file web/adapter/json.go
type jsonEncoder struct{}

// SetOption does nothing because this encoder does not recognize any option.
func (je *jsonEncoder) SetOption(option encoder.Option) {}

// WriteZettel writes the encoded zettel to the writer.
func (je *jsonEncoder) WriteZettel(
	w io.Writer, zn *ast.ZettelNode, inhMeta bool) (int, error) {
	return 0, encoder.ErrNoWriteZettel
}

// WriteMeta encodes meta data as HTML5.
func (je *jsonEncoder) WriteMeta(w io.Writer, meta *meta.Meta) (int, error) {
	return 0, encoder.ErrNoWriteMeta
}

func (je *jsonEncoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return 0, encoder.ErrNoWriteContent
}

// WriteBlocks writes a block slice to the writer
func (je *jsonEncoder) WriteBlocks(w io.Writer, bs ast.BlockSlice) (int, error) {
	return 0, encoder.ErrNoWriteBlocks
}

// WriteInlines writes an inline slice to the writer
func (je *jsonEncoder) WriteInlines(w io.Writer, is ast.InlineSlice) (int, error) {
	return 0, encoder.ErrNoWriteInlines
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
