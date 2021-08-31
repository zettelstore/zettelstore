//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package htmlenc encodes the abstract syntax tree into HTML5.
package htmlenc

import (
	"io"

	"zettelstore.de/z/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
)

func init() {
	encoder.Register(api.EncoderHTML, encoder.Info{
		Create: func(env *encoder.Environment) encoder.Encoder { return &htmlEncoder{env: env} },
	})
}

type htmlEncoder struct {
	env *encoder.Environment
}

// WriteZettel encodes a full zettel as HTML5.
func (he *htmlEncoder) WriteZettel(w io.Writer, zn *ast.ZettelNode, evalMeta encoder.EvalMetaFunc) (int, error) {
	v := newVisitor(he, w)
	if !he.env.IsXHTML() {
		v.b.WriteString("<!DOCTYPE html>\n")
	}
	if env := he.env; env != nil && env.Lang == "" {
		v.b.WriteStrings("<html>\n<head>")
	} else {
		v.b.WriteStrings("<html lang=\"", env.Lang, "\">")
	}
	v.b.WriteString("\n<head>\n<meta charset=\"utf-8\">\n")
	v.b.WriteStrings("<title>", v.evalValue(zn.InhMeta.GetDefault(meta.KeyTitle, ""), evalMeta), "</title>")
	v.acceptMeta(zn.InhMeta, evalMeta)
	v.b.WriteString("\n</head>\n<body>\n")
	ast.Walk(v, zn.Ast)
	v.writeEndnotes()
	v.b.WriteString("</body>\n</html>")
	length, err := v.b.Flush()
	return length, err
}

// WriteMeta encodes meta data as HTML5.
func (he *htmlEncoder) WriteMeta(w io.Writer, m *meta.Meta, evalMeta encoder.EvalMetaFunc) (int, error) {
	v := newVisitor(he, w)

	// Write title
	if title, ok := m.Get(meta.KeyTitle); ok {
		v.b.WriteStrings("<meta name=\"zs-", meta.KeyTitle, "\" content=\"")
		v.writeQuotedEscaped(v.evalValue(title, evalMeta))
		v.b.WriteString("\">")
	}

	// Write other metadata
	v.acceptMeta(m, evalMeta)
	length, err := v.b.Flush()
	return length, err
}

func (he *htmlEncoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return he.WriteBlocks(w, zn.Ast)
}

// WriteBlocks encodes a block slice.
func (he *htmlEncoder) WriteBlocks(w io.Writer, bln *ast.BlockListNode) (int, error) {
	v := newVisitor(he, w)
	ast.Walk(v, bln)
	v.writeEndnotes()
	length, err := v.b.Flush()
	return length, err
}

// WriteInlines writes an inline slice to the writer
func (he *htmlEncoder) WriteInlines(w io.Writer, iln *ast.InlineListNode) (int, error) {
	v := newVisitor(he, w)
	if env := he.env; env != nil {
		v.inInteractive = env.Interactive
	}
	ast.Walk(v, iln)
	length, err := v.b.Flush()
	return length, err
}
