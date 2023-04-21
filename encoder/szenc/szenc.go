//-----------------------------------------------------------------------------
// Copyright (c) 2022-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package szenc encodes the abstract syntax tree into a s-expr for zettel.
package szenc

import (
	"io"

	"codeberg.org/t73fde/sxpf"
	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
)

func init() {
	encoder.Register(api.EncoderSz, func() encoder.Encoder { return Create() })
}

// Create a S-expr encoder
func Create() *Encoder { return &mySE }

type Encoder struct{}

var mySE Encoder

// WriteZettel writes the encoded zettel to the writer.
func (*Encoder) WriteZettel(w io.Writer, zn *ast.ZettelNode, evalMeta encoder.EvalMetaFunc) (int, error) {
	t := NewTransformer()
	content := t.GetSz(&zn.Ast)
	meta := t.GetMeta(zn.InhMeta, evalMeta)
	return io.WriteString(w, sxpf.Nil().Cons(content).Cons(meta).Repr())
}

// WriteMeta encodes meta data as s-expression.
func (*Encoder) WriteMeta(w io.Writer, m *meta.Meta, evalMeta encoder.EvalMetaFunc) (int, error) {
	t := NewTransformer()
	return io.WriteString(w, t.GetMeta(m, evalMeta).Repr())
}

func (se *Encoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return se.WriteBlocks(w, &zn.Ast)
}

// WriteBlocks writes a block slice to the writer
func (*Encoder) WriteBlocks(w io.Writer, bs *ast.BlockSlice) (int, error) {
	t := NewTransformer()
	return io.WriteString(w, t.GetSz(bs).Repr())
}

// WriteInlines writes an inline slice to the writer
func (*Encoder) WriteInlines(w io.Writer, is *ast.InlineSlice) (int, error) {
	t := NewTransformer()
	return io.WriteString(w, t.GetSz(is).Repr())
}
