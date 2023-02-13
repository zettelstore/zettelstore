//-----------------------------------------------------------------------------
// Copyright (c) 2023-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package shtmlenc encodes the abstract syntax tree into a s-expr which represents HTML.
package shtmlenc

import (
	"io"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
)

func init() {
	encoder.Register(api.EncoderSHTML, func() encoder.Encoder { return Create() })
}

// Create a SHTML encoder
func Create() *Encoder { return &mySE }

type Encoder struct{}

var mySE Encoder

// WriteZettel writes the encoded zettel to the writer.
func (*Encoder) WriteZettel(w io.Writer, zn *ast.ZettelNode, evalMeta encoder.EvalMetaFunc) (int, error) {
	return 0, nil
}

// WriteMeta encodes meta data as s-expression.
func (*Encoder) WriteMeta(w io.Writer, m *meta.Meta, evalMeta encoder.EvalMetaFunc) (int, error) {
	return 0, nil
}

func (se *Encoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return se.WriteBlocks(w, &zn.Ast)
}

// WriteBlocks writes a block slice to the writer
func (*Encoder) WriteBlocks(w io.Writer, bs *ast.BlockSlice) (int, error) {
	if len(*bs) == 0 {
		return io.WriteString(w, "()")
	}
	return 0, nil
}

// WriteInlines writes an inline slice to the writer
func (*Encoder) WriteInlines(w io.Writer, is *ast.InlineSlice) (int, error) {
	if len(*is) == 0 {
		return io.WriteString(w, "()")
	}
	return 0, nil
}
