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
	"strings"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/parser"
)

func init() {
	encoder.Register("html", encoder.Info{
		Create: func(env *encoder.Environment) encoder.Encoder { return &htmlEncoder{env: env} },
	})
}

type htmlEncoder struct {
	env *encoder.Environment
}

// WriteZettel encodes a full zettel as HTML5.
func (he *htmlEncoder) WriteZettel(
	w io.Writer, zn *ast.ZettelNode, inhMeta bool) (int, error) {
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
	textEnc := encoder.Create("text", nil)
	var sb strings.Builder
	textEnc.WriteInlines(&sb, zn.Title)
	v.b.WriteStrings("<title>", sb.String(), "</title>")
	if inhMeta {
		v.acceptMeta(zn.InhMeta)
	} else {
		v.acceptMeta(zn.Zettel.Meta)
	}
	v.b.WriteString("\n</head>\n<body>\n")
	v.acceptBlockSlice(zn.Ast)
	v.writeEndnotes()
	v.b.WriteString("</body>\n</html>")
	length, err := v.b.Flush()
	return length, err
}

// WriteMeta encodes meta data as HTML5.
func (he *htmlEncoder) WriteMeta(w io.Writer, m *meta.Meta) (int, error) {
	v := newVisitor(he, w)

	// Write title
	if title, ok := m.Get(meta.KeyTitle); ok {
		textEnc := encoder.Create("text", nil)
		var sb strings.Builder
		textEnc.WriteInlines(&sb, parser.ParseMetadata(title))
		v.b.WriteStrings("<meta name=\"zs-", meta.KeyTitle, "\" content=\"")
		v.writeQuotedEscaped(sb.String())
		v.b.WriteString("\">")
	}

	// Write other metadata
	v.acceptMeta(m)
	length, err := v.b.Flush()
	return length, err
}

func (he *htmlEncoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return he.WriteBlocks(w, zn.Ast)
}

// WriteBlocks encodes a block slice.
func (he *htmlEncoder) WriteBlocks(w io.Writer, bs ast.BlockSlice) (int, error) {
	v := newVisitor(he, w)
	v.acceptBlockSlice(bs)
	v.writeEndnotes()
	length, err := v.b.Flush()
	return length, err
}

// WriteInlines writes an inline slice to the writer
func (he *htmlEncoder) WriteInlines(w io.Writer, is ast.InlineSlice) (int, error) {
	v := newVisitor(he, w)
	v.acceptInlineSlice(is)
	length, err := v.b.Flush()
	return length, err
}
