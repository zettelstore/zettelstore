//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package rawenc encodes the abstract syntax tree as raw content.
package rawenc

import (
	"io"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
)

func init() {
	encoder.Register("raw", encoder.Info{
		Create: func(*encoder.Environment) encoder.Encoder { return &rawEncoder{} },
	})
}

type rawEncoder struct{}

// WriteZettel writes the encoded zettel to the writer.
func (re *rawEncoder) WriteZettel(
	w io.Writer, zn *ast.ZettelNode, inhMeta bool) (int, error) {
	b := encoder.NewBufWriter(w)
	if inhMeta {
		zn.InhMeta.Write(&b, true)
	} else {
		zn.Meta.Write(&b, true)
	}
	b.WriteByte('\n')
	b.WriteString(zn.Content.AsString())
	length, err := b.Flush()
	return length, err
}

// WriteMeta encodes meta data as HTML5.
func (re *rawEncoder) WriteMeta(w io.Writer, m *meta.Meta) (int, error) {
	b := encoder.NewBufWriter(w)
	m.Write(&b, true)
	length, err := b.Flush()
	return length, err
}

func (re *rawEncoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	b := encoder.NewBufWriter(w)
	b.WriteString(zn.Content.AsString())
	length, err := b.Flush()
	return length, err
}

// WriteBlocks writes a block slice to the writer
func (re *rawEncoder) WriteBlocks(w io.Writer, bs ast.BlockSlice) (int, error) {
	return 0, encoder.ErrNoWriteBlocks
}

// WriteInlines writes an inline slice to the writer
func (re *rawEncoder) WriteInlines(w io.Writer, is ast.InlineSlice) (int, error) {
	return 0, encoder.ErrNoWriteInlines
}
