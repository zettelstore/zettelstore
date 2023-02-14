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
	"zettelstore.de/z/encoder/sexprenc"
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
	tx := sexprenc.NewTransformer()
	th := shtml.NewTransformer(1)
	metaSHTML, err := th.Transform(tx.GetMeta(zn.InhMeta, evalMeta))
	if err != nil {
		return 0, err
	}
	contentSHTML, err := th.Transform(tx.GetSexpr(&zn.Ast))
	if err != nil {
		return 0, err
	}
	result := sxpf.MakePair(metaSHTML, contentSHTML)
	return result.Print(w)
}

// WriteMeta encodes meta data as s-expression.
func (*Encoder) WriteMeta(w io.Writer, m *meta.Meta, evalMeta encoder.EvalMetaFunc) (int, error) {
	tx := sexprenc.NewTransformer()
	th := shtml.NewTransformer(1)
	metaSHTML, err := th.Transform(tx.GetMeta(m, evalMeta))
	if err != nil {
		return 0, err
	}
	return metaSHTML.Print(w)
}

func (se *Encoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return se.WriteBlocks(w, &zn.Ast)
}

// WriteBlocks writes a block slice to the writer
func (*Encoder) WriteBlocks(w io.Writer, bs *ast.BlockSlice) (int, error) {
	hval, err := TransformSlice(bs)
	if err != nil {
		return 0, err
	}
	return hval.Print(w)
}

// WriteInlines writes an inline slice to the writer
func (*Encoder) WriteInlines(w io.Writer, is *ast.InlineSlice) (int, error) {
	hval, err := TransformSlice(is)
	if err != nil {
		return 0, err
	}
	return hval.Print(w)
}

// TransformSlice transforms a AST slice into SHTML.
func TransformSlice(node ast.Node) (*sxpf.List, error) {
	tx := sexprenc.NewTransformer()
	xval := tx.GetSexpr(node)
	th := shtml.NewTransformer(1)
	return th.Transform(xval)
}
