//-----------------------------------------------------------------------------
// Copyright (c) 2022-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2022-present Detlef Stern
//-----------------------------------------------------------------------------

// Package htmlenc encodes the abstract syntax tree into HTML5 via zettelstore-client.
package htmlenc

import (
	"io"
	"strings"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/client.fossil/shtml"
	"zettelstore.de/sx.fossil"
	"zettelstore.de/sx.fossil/sxhtml"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/encoder/szenc"
	"zettelstore.de/z/encoder/textenc"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/zettel/meta"
)

func init() {
	encoder.Register(api.EncoderHTML, func(params *encoder.CreateParameter) encoder.Encoder { return Create(params) })
}

// Create an encoder.
func Create(params *encoder.CreateParameter) *Encoder {
	// We need a new transformer every time, because tx.inVerse must be unique.
	// If we can refactor it out, the transformer can be created only once.
	return &Encoder{
		tx:      szenc.NewTransformer(),
		th:      shtml.NewEvaluator(1),
		lang:    params.Lang,
		textEnc: textenc.Create(),
	}
}

type Encoder struct {
	tx      *szenc.Transformer
	th      *shtml.Evaluator
	lang    string
	textEnc *textenc.Encoder
}

// WriteZettel encodes a full zettel as HTML5.
func (he *Encoder) WriteZettel(w io.Writer, zn *ast.ZettelNode, evalMeta encoder.EvalMetaFunc) (int, error) {
	env := shtml.MakeEnvironment(he.lang)
	hm, err := he.th.Evaluate(he.tx.GetMeta(zn.InhMeta, evalMeta), &env)
	if err != nil {
		return 0, err
	}

	var isTitle ast.InlineSlice
	var htitle *sx.Pair
	plainTitle, hasTitle := zn.InhMeta.Get(api.KeyTitle)
	if hasTitle {
		isTitle = parser.ParseSpacedText(plainTitle)
		xtitle := he.tx.GetSz(&isTitle)
		htitle, err = he.th.Evaluate(xtitle, &env)
		if err != nil {
			return 0, err
		}
	}

	xast := he.tx.GetSz(&zn.Ast)
	hast, err := he.th.Evaluate(xast, &env)
	if err != nil {
		return 0, err
	}
	hen := he.th.Endnotes(&env)

	head := sx.MakeList(shtml.SymHead)
	curr := head
	curr = curr.AppendBang(sx.Nil().Cons(sx.Nil().Cons(sx.Cons(sx.Symbol("charset"), sx.String("utf-8"))).Cons(sxhtml.SymAttr)).Cons(shtml.SymMeta))
	for elem := hm; elem != nil; elem = elem.Tail() {
		curr = curr.AppendBang(elem.Car())
	}
	var sb strings.Builder
	if hasTitle {
		he.textEnc.WriteInlines(&sb, &isTitle)
	} else {
		sb.Write(zn.Meta.Zid.Bytes())
	}
	_ = curr.AppendBang(sx.Nil().Cons(sx.String(sb.String())).Cons(shtml.SymAttrTitle))

	body := sx.MakeList(shtml.SymBody)
	curr = body
	if hasTitle {
		curr = curr.AppendBang(htitle.Cons(shtml.SymH1))
	}
	for elem := hast; elem != nil; elem = elem.Tail() {
		curr = curr.AppendBang(elem.Car())
	}
	if hen != nil {
		curr = curr.AppendBang(sx.Nil().Cons(shtml.SymHR))
		_ = curr.AppendBang(hen)
	}

	doc := sx.MakeList(
		sxhtml.SymDoctype,
		sx.MakeList(shtml.SymHtml, head, body),
	)

	gen := sxhtml.NewGenerator(sxhtml.WithNewline)
	return gen.WriteHTML(w, doc)
}

// WriteMeta encodes meta data as HTML5.
func (he *Encoder) WriteMeta(w io.Writer, m *meta.Meta, evalMeta encoder.EvalMetaFunc) (int, error) {
	env := shtml.MakeEnvironment(he.lang)
	hm, err := he.th.Evaluate(he.tx.GetMeta(m, evalMeta), &env)
	if err != nil {
		return 0, err
	}
	gen := sxhtml.NewGenerator(sxhtml.WithNewline)
	return gen.WriteListHTML(w, hm)
}

func (he *Encoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return he.WriteBlocks(w, &zn.Ast)
}

// WriteBlocks encodes a block slice.
func (he *Encoder) WriteBlocks(w io.Writer, bs *ast.BlockSlice) (int, error) {
	env := shtml.MakeEnvironment(he.lang)
	hobj, err := he.th.Evaluate(he.tx.GetSz(bs), &env)
	if err == nil {
		gen := sxhtml.NewGenerator()
		length, err2 := gen.WriteListHTML(w, hobj)
		if err2 != nil {
			return length, err2
		}

		l, err2 := gen.WriteHTML(w, he.th.Endnotes(&env))
		length += l
		return length, err2
	}
	return 0, err
}

// WriteInlines writes an inline slice to the writer
func (he *Encoder) WriteInlines(w io.Writer, is *ast.InlineSlice) (int, error) {
	env := shtml.MakeEnvironment(he.lang)
	hobj, err := he.th.Evaluate(he.tx.GetSz(is), &env)
	if err == nil {
		gen := sxhtml.NewGenerator()
		length, err2 := gen.WriteListHTML(w, hobj)
		if err2 != nil {
			return length, err2
		}
		return length, nil
	}
	return 0, err
}
