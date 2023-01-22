//-----------------------------------------------------------------------------
// Copyright (c) 2022-2023 Detlef Stern
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

	"codeberg.org/t73fde/sxpf"
	"zettelstore.de/c/api"
	"zettelstore.de/c/html"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/encoder/sexprenc"
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
	io.WriteString(w, "<html>\n<head>\n<meta charset=\"utf-8\">\n")
	plainTitle, hasTitle := zn.InhMeta.Get(api.KeyTitle)
	if hasTitle {
		io.WriteString(w, "<title>")
		is := evalMeta(plainTitle)
		he.textEnc.WriteInlines(w, &is)
		io.WriteString(w, "</title>\n")
	}

	acceptMeta(w, he.textEnc, zn.InhMeta, evalMeta)
	io.WriteString(w, "</head>\n<body>\n")
	env := html.NewEncEnvironment(w, 1)
	if hasTitle {
		if isTitle := evalMeta(plainTitle); len(isTitle) > 0 {
			io.WriteString(w, "<h1>")
			if err := acceptInlines(env, &isTitle); err != nil {
				return 0, err
			}
			io.WriteString(w, "</h1>\n")
		}
	}

	err := acceptBlocks(env, &zn.Ast)
	if err == nil {
		// env.WriteEndnotes()
		io.WriteString(w, "</body>\n</html>")
	}
	return 0, err
}

// WriteMeta encodes meta data as HTML5.
func (he *Encoder) WriteMeta(w io.Writer, m *meta.Meta, evalMeta encoder.EvalMetaFunc) (int, error) {
	acceptMeta(w, he.textEnc, m, evalMeta)
	return 0, nil
}

func (he *Encoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return he.WriteBlocks(w, &zn.Ast)
}

// WriteBlocks encodes a block slice.
func (*Encoder) WriteBlocks(w io.Writer, bs *ast.BlockSlice) (int, error) {
	env := html.NewEncEnvironment(w, 1)
	err := acceptBlocks(env, bs)
	if err == nil {
		env.WriteEndnotes()
		err = env.GetError()
	}
	return 0, err
}

// WriteInlines writes an inline slice to the writer
func (*Encoder) WriteInlines(w io.Writer, is *ast.InlineSlice) (int, error) {
	env := html.NewEncEnvironment(w, 1)
	return 0, acceptInlines(env, is)
}

func acceptMeta(w io.Writer, textEnc encoder.Encoder, m *meta.Meta, evalMeta encoder.EvalMetaFunc) {
	for _, p := range m.ComputedPairs() {
		io.WriteString(w, `<meta name="zs-`)
		io.WriteString(w, p.Key)
		io.WriteString(w, `" content="`)
		is := evalMeta(p.Value)
		var sb strings.Builder
		textEnc.WriteInlines(&sb, &is)
		html.AttributeEscape(w, sb.String())
		io.WriteString(w, "\">\n")
	}
}

func acceptBlocks(env *html.EncEnvironment, bs *ast.BlockSlice) error {
	lst := sexprenc.GetSexpr(bs)
	sxpf.Eval(env, lst)
	return env.GetError()
}
func acceptInlines(env *html.EncEnvironment, is *ast.InlineSlice) error {
	lst := sexprenc.GetSexpr(is)
	sxpf.Eval(env, lst)
	return env.GetError()
}
