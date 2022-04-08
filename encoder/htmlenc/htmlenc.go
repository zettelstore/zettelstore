//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
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
	"bytes"
	"io"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/c/html"
	"zettelstore.de/c/zjson"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
)

func init() {
	encoder.Register(api.EncoderHTML, encoder.Info{
		Create: func(env *encoder.Environment) encoder.Encoder {
			return &chtmlEncoder{
				env:      env,
				zjsonEnc: encoder.Create(api.EncoderZJSON, nil),
				textEnc:  encoder.Create(api.EncoderText, nil),
			}
		},
	})
}

type chtmlEncoder struct {
	env      *encoder.Environment
	zjsonEnc encoder.Encoder
	textEnc  encoder.Encoder
}

// WriteZettel encodes a full zettel as HTML5.
func (he *chtmlEncoder) WriteZettel(w io.Writer, zn *ast.ZettelNode, evalMeta encoder.EvalMetaFunc) (int, error) {
	if env := he.env; env != nil && env.Lang == "" {
		io.WriteString(w, "<html>")
	} else {
		io.WriteString(w, `<html lang="`)
		io.WriteString(w, env.Lang)
		io.WriteString(w, `">`)
	}
	io.WriteString(w, "\n<head>\n<meta charset=\"utf-8\">\n")
	plainTitle, hasTitle := zn.InhMeta.Get(api.KeyTitle)
	if hasTitle {
		io.WriteString(w, "<title>")
		is := evalMeta(plainTitle)
		he.textEnc.WriteInlines(w, &is)
		io.WriteString(w, "</title>\n")
	}

	he.acceptMeta(w, zn.InhMeta, evalMeta)
	io.WriteString(w, "</head>\n<body>\n")
	enc := html.NewEncoder(w, 1)
	if hasTitle {
		if isTitle := evalMeta(plainTitle); len(isTitle) > 0 {
			io.WriteString(w, "<h1>")
			if err := he.acceptInlines(enc, &isTitle); err != nil {
				return 0, err
			}
			io.WriteString(w, "</h1>\n")
		}
	}

	var buf bytes.Buffer
	_, err := he.zjsonEnc.WriteBlocks(&buf, &zn.Ast)
	if err != nil {
		return 0, err
	}
	val, err := zjson.Decode(&buf)
	if err != nil {
		return 0, err
	}
	enc.TraverseBlock(zjson.MakeArray(val))
	enc.WriteEndnotes()
	io.WriteString(w, "</body>\n</html>")
	return 0, nil
}

// WriteMeta encodes meta data as HTML5.
func (he *chtmlEncoder) WriteMeta(w io.Writer, m *meta.Meta, evalMeta encoder.EvalMetaFunc) (int, error) {
	he.acceptMeta(w, m, evalMeta)
	return 0, nil
}

func (he *chtmlEncoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return he.WriteBlocks(w, &zn.Ast)
}

// WriteBlocks encodes a block slice.
func (he *chtmlEncoder) WriteBlocks(w io.Writer, bs *ast.BlockSlice) (int, error) {
	var buf bytes.Buffer
	_, err := he.zjsonEnc.WriteBlocks(&buf, bs)
	if err != nil {
		return 0, err
	}
	val, err := zjson.Decode(&buf)
	if err != nil {
		return 0, err
	}
	enc := html.NewEncoder(w, 1)
	enc.TraverseBlock(zjson.MakeArray(val))
	enc.WriteEndnotes()
	return 0, nil
}

// WriteInlines writes an inline slice to the writer
func (he *chtmlEncoder) WriteInlines(w io.Writer, is *ast.InlineSlice) (int, error) {
	enc := html.NewEncoder(w, 1)
	if err := he.acceptInlines(enc, is); err != nil {
		return 0, err
	}
	return 0, nil
}

func (he *chtmlEncoder) acceptMeta(w io.Writer, m *meta.Meta, evalMeta encoder.EvalMetaFunc) {
	for _, p := range m.ComputedPairs() {
		io.WriteString(w, `<meta name="zs-`)
		io.WriteString(w, p.Key)
		io.WriteString(w, `" content="`)
		is := evalMeta(p.Value)
		var sb strings.Builder
		he.textEnc.WriteInlines(&sb, &is)
		html.AttributeEscape(w, sb.String())
		io.WriteString(w, "\">\n")
	}
}

func (he *chtmlEncoder) acceptInlines(enc *html.Encoder, is *ast.InlineSlice) error {
	var buf bytes.Buffer
	_, err := he.zjsonEnc.WriteInlines(&buf, is)
	if err != nil {
		return err
	}
	val, err := zjson.Decode(&buf)
	if err != nil {
		return err
	}
	enc.TraverseInline(zjson.MakeArray(val))
	return nil
}
