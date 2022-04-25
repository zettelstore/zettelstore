//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package sexprenc encodes the abstract syntax tree into a s-expr.
package sexprenc

import (
	"io"

	"zettelstore.de/c/api"
	"zettelstore.de/c/sexpr"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
)

func init() {
	encoder.Register(api.EncoderSexpr, func() encoder.Encoder { return Create() })
}

// Create a S-expr encoder
func Create() *Encoder { return &mySE }

type Encoder struct{}

var mySE Encoder

// WriteZettel writes the encoded zettel to the writer.
func (*Encoder) WriteZettel(w io.Writer, zn *ast.ZettelNode, evalMeta encoder.EvalMetaFunc) (int, error) {
	content := GetSexpr(&zn.Ast)
	meta := GetMeta(zn.InhMeta, evalMeta)
	return sexpr.NewList(content, meta).Encode(w)
}

// WriteMeta encodes meta data as JSON.
func (*Encoder) WriteMeta(w io.Writer, m *meta.Meta, evalMeta encoder.EvalMetaFunc) (int, error) {
	return GetMeta(m, evalMeta).Encode(w)
}

func (se *Encoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return se.WriteBlocks(w, &zn.Ast)
}

// WriteBlocks writes a block slice to the writer
func (*Encoder) WriteBlocks(w io.Writer, bs *ast.BlockSlice) (int, error) {
	return GetSexpr(bs).Encode(w)
}

// WriteInlines writes an inline slice to the writer
func (*Encoder) WriteInlines(w io.Writer, is *ast.InlineSlice) (int, error) {
	return GetSexpr(is).Encode(w)
}
