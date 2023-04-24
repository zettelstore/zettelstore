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

	"codeberg.org/t73fde/sxpf"
	"zettelstore.de/c/api"
	"zettelstore.de/c/shtml"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/encoder/szenc"
)

func init() {
	encoder.Register(api.EncoderSHTML, func() encoder.Encoder { return Create() })
}

// Create a SHTML encoder
func Create() *Encoder {
	// We need a new transformer every time, because tx.inVerse must be unique.
	// If we can refactor it out, the transformer can be created only once.
	return &Encoder{
		tx: szenc.NewTransformer(),
		th: shtml.NewTransformer(1, nil),
	}
}

type Encoder struct {
	tx *szenc.Transformer
	th *shtml.Transformer
}

// WriteZettel writes the encoded zettel to the writer.
func (enc *Encoder) WriteZettel(w io.Writer, zn *ast.ZettelNode, evalMeta encoder.EvalMetaFunc) (int, error) {
	metaSHTML, err := enc.th.Transform(enc.tx.GetMeta(zn.InhMeta, evalMeta))
	if err != nil {
		return 0, err
	}
	contentSHTML, err := enc.th.Transform(enc.tx.GetSz(&zn.Ast))
	if err != nil {
		return 0, err
	}
	result := sxpf.Cons(metaSHTML, contentSHTML)
	return result.Print(w)
}

// WriteMeta encodes meta data as s-expression.
func (enc *Encoder) WriteMeta(w io.Writer, m *meta.Meta, evalMeta encoder.EvalMetaFunc) (int, error) {
	metaSHTML, err := enc.th.Transform(enc.tx.GetMeta(m, evalMeta))
	if err != nil {
		return 0, err
	}
	return metaSHTML.Print(w)
}

func (enc *Encoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return enc.WriteBlocks(w, &zn.Ast)
}

// WriteBlocks writes a block slice to the writer
func (enc *Encoder) WriteBlocks(w io.Writer, bs *ast.BlockSlice) (int, error) {
	hval, err := enc.th.Transform(enc.tx.GetSz(bs))
	if err != nil {
		return 0, err
	}
	return hval.Print(w)
}

// WriteInlines writes an inline slice to the writer
func (enc *Encoder) WriteInlines(w io.Writer, is *ast.InlineSlice) (int, error) {
	hval, err := enc.th.Transform(enc.tx.GetSz(is))
	if err != nil {
		return 0, err
	}
	return hval.Print(w)
}
