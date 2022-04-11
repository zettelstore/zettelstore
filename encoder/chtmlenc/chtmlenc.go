//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package chtmlenc encodes the abstract syntax tree into HTML5 (deprecated).
// It is only used for the WebUI and will be deprecated by a software that
// uses the zettelstore-client HTML encoder.
package chtmlenc

import (
	"io"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
)

// Create a new encoder.
func Create(env *encoder.Environment) encoder.Encoder { return &htmlEncoder{env: env} }

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
	plainTitle, hasTitle := zn.InhMeta.Get(api.KeyTitle)
	if hasTitle {
		v.b.WriteStrings("<title>", v.evalValue(plainTitle, evalMeta), "</title>")
	}
	v.acceptMeta(zn.InhMeta, evalMeta)
	v.b.WriteString("\n</head>\n<body>\n")
	if hasTitle {
		if isTitle := evalMeta(plainTitle); len(isTitle) > 0 {
			v.b.WriteString("<h1>")
			ast.Walk(v, &isTitle)
			v.b.WriteString("</h1>\n")
		}
	}
	ast.Walk(v, &zn.Ast)
	v.writeEndnotes()
	v.b.WriteString("</body>\n</html>")
	length, err := v.b.Flush()
	return length, err
}

// WriteMeta encodes meta data as HTML5.
func (he *htmlEncoder) WriteMeta(w io.Writer, m *meta.Meta, evalMeta encoder.EvalMetaFunc) (int, error) {
	v := newVisitor(he, w)

	// Write title
	if title, ok := m.Get(api.KeyTitle); ok {
		v.b.WriteStrings("<meta name=\"zs-", api.KeyTitle, "\" content=\"")
		v.writeQuotedEscaped(v.evalValue(title, evalMeta))
		v.b.WriteString("\">")
	}

	// Write other metadata
	v.acceptMeta(m, evalMeta)
	length, err := v.b.Flush()
	return length, err
}

func (he *htmlEncoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return he.WriteBlocks(w, &zn.Ast)
}

// WriteBlocks encodes a block slice.
func (he *htmlEncoder) WriteBlocks(w io.Writer, bs *ast.BlockSlice) (int, error) {
	v := newVisitor(he, w)
	ast.Walk(v, bs)
	v.writeEndnotes()
	length, err := v.b.Flush()
	return length, err
}

// WriteInlines writes an inline slice to the writer
func (he *htmlEncoder) WriteInlines(w io.Writer, is *ast.InlineSlice) (int, error) {
	v := newVisitor(he, w)
	if env := he.env; env != nil {
		v.inInteractive = env.Interactive
	}
	ast.Walk(v, is)
	length, err := v.b.Flush()
	return length, err
}
