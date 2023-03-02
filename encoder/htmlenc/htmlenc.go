//-----------------------------------------------------------------------------
// Copyright (c) 2022-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package htmlenc encodes the abstract syntax tree into HTML5 via zettelstore-client.
package htmlenc

import (
	"io"
	"strings"

	"codeberg.org/t73fde/sxhtml"
	"codeberg.org/t73fde/sxpf"
	"zettelstore.de/c/api"
	"zettelstore.de/c/shtml"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/encoder/sexprenc"
	"zettelstore.de/z/encoder/shtmlenc"
	"zettelstore.de/z/encoder/textenc"
)

func init() {
	encoder.Register(api.EncoderHTML, func() encoder.Encoder { return Create() })
}

// Create an encoder.
func Create() *Encoder { return &mySHE }

type Encoder struct {
	textEnc *textenc.Encoder
}

var mySHE = Encoder{
	textEnc: textenc.Create(),
}

// WriteZettel encodes a full zettel as HTML5.
func (he *Encoder) WriteZettel(w io.Writer, zn *ast.ZettelNode, evalMeta encoder.EvalMetaFunc) (int, error) {
	tx := sexprenc.NewTransformer()
	xm := tx.GetMeta(zn.InhMeta, evalMeta)

	th := shtml.NewTransformer(1, nil)
	hm, err := th.Transform(xm)
	if err != nil {
		return 0, err
	}

	plainTitle, hasTitle := zn.InhMeta.Get(api.KeyTitle)
	evalTitle := evalMeta(plainTitle)
	var htitle *sxpf.List
	if hasTitle {
		xtitle := tx.GetSexpr(&evalTitle)
		htitle, err = th.Transform(xtitle)
		if err != nil {
			return 0, err
		}
	}

	xast := tx.GetSexpr(&zn.Ast)
	hast, err := th.Transform(xast)
	if err != nil {
		return 0, err
	}
	hen := th.Endnotes()

	sf := th.SymbolFactory()
	symAttr := sf.MustMake(sxhtml.NameSymAttr)

	head := sxpf.Nil().Cons(sf.MustMake("head"))
	curr := head
	curr = curr.AppendBang(sxpf.Nil().Cons(sxpf.Nil().Cons(sxpf.Cons(sf.MustMake("charset"), sxpf.MakeString("utf-8"))).Cons(symAttr)).Cons(sf.MustMake("meta")))
	for elem := hm; elem != nil; elem = elem.Tail() {
		curr = curr.AppendBang(elem.Car())
	}
	if hasTitle {
		var sb strings.Builder
		he.textEnc.WriteInlines(&sb, &evalTitle)
		_ = curr.AppendBang(sxpf.Nil().Cons(sxpf.MakeString(sb.String())).Cons(sf.MustMake("title")))
	}

	body := sxpf.Nil().Cons(sf.MustMake("body"))
	curr = body
	if hasTitle {
		curr = curr.AppendBang(htitle.Cons(sf.MustMake("h1")))
	}
	for elem := hast; elem != nil; elem = elem.Tail() {
		curr = curr.AppendBang(elem.Car())
	}
	if hen != nil {
		curr = curr.AppendBang(sxpf.Nil().Cons(sf.MustMake("hr")))
		_ = curr.AppendBang(hen)
	}

	doc := sxpf.MakeList(
		sf.MustMake(sxhtml.NameSymDoctype),
		sxpf.MakeList(sf.MustMake("html"), head, body),
	)

	gen := sxhtml.NewGenerator(sf, sxhtml.WithNewline)
	return gen.WriteHTML(w, doc)
}

// WriteMeta encodes meta data as HTML5.
func (he *Encoder) WriteMeta(w io.Writer, m *meta.Meta, evalMeta encoder.EvalMetaFunc) (int, error) {
	tx := sexprenc.NewTransformer()
	xm := tx.GetMeta(m, evalMeta)

	th := shtml.NewTransformer(1, nil)
	hm, err := th.Transform(xm)
	if err != nil {
		return 0, err
	}
	gen := sxhtml.NewGenerator(th.SymbolFactory(), sxhtml.WithNewline)
	return gen.WriteListHTML(w, hm)
}

func (he *Encoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return he.WriteBlocks(w, &zn.Ast)
}

// WriteBlocks encodes a block slice.
func (*Encoder) WriteBlocks(w io.Writer, bs *ast.BlockSlice) (int, error) {
	tx := sexprenc.NewTransformer()
	xval := tx.GetSexpr(bs)
	th := shtml.NewTransformer(1, nil)
	hobj, err := th.Transform(xval)

	if err == nil {
		gen := sxhtml.NewGenerator(th.SymbolFactory())
		length, err2 := gen.WriteListHTML(w, hobj)
		if err2 != nil {
			return length, err2
		}

		l, err2 := gen.WriteHTML(w, th.Endnotes())
		length += l
		return length, err2
	}
	return 0, err
}

// WriteInlines writes an inline slice to the writer
func (*Encoder) WriteInlines(w io.Writer, is *ast.InlineSlice) (int, error) {
	hobj, err := shtmlenc.TransformSlice(is)
	if err == nil {
		gen := sxhtml.NewGenerator(sxpf.FindSymbolFactory(hobj))
		length, err2 := gen.WriteListHTML(w, hobj)
		if err2 != nil {
			return length, err2
		}
		return length, nil
	}
	return 0, err
}
