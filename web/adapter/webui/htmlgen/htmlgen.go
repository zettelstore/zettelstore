//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package htmlgen encodes the abstract syntax tree into HTML5 (deprecated).
// It is only used for the WebUI and will be deprecated by a software that
// uses the zettelstore-client HTML encoder.
package htmlgen

import (
	"io"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/strfun"
)

// Create a new encoder.
func Create(lang, extMarker string, interactive, newWindow bool, ignoreMeta strfun.Set) *Encoder {
	return &Encoder{lang: lang, extMarker: extMarker, interactive: interactive, newWindow: newWindow, ignoreMeta: ignoreMeta}
}

// Encoder encapsulates the encoder itself
type Encoder struct {
	lang        string
	extMarker   string
	interactive bool
	newWindow   bool
	ignoreMeta  strfun.Set
	footnotes   []footnoteInfo // Stores footnotes detected while encoding
	footnoteNum int
}
type footnoteInfo struct {
	fn  *ast.FootnoteNode
	num int
}

// addFootnote adds a footnote node to the environment and returns the number of that footnote.
func (he *Encoder) addFootnote(fn *ast.FootnoteNode) int {
	he.footnoteNum++
	he.footnotes = append(he.footnotes, footnoteInfo{fn: fn, num: he.footnoteNum})
	return he.footnoteNum
}

// popFootnote returns the next footnote and removes it from the list.
func (he *Encoder) popFootnote() (*ast.FootnoteNode, int) {
	if len(he.footnotes) == 0 {
		he.footnotes = nil
		he.footnoteNum = 0
		return nil, -1
	}
	fni := he.footnotes[0]
	he.footnotes = he.footnotes[1:]
	return fni.fn, fni.num
}

// WriteZettel encodes a full zettel as HTML5.
func (he *Encoder) WriteZettel(w io.Writer, zn *ast.ZettelNode, evalMeta encoder.EvalMetaFunc) (int, error) {
	v := newVisitor(he, w)

	if he.lang == "" {
		v.b.WriteStrings("<html>\n<head>")
	} else {
		v.b.WriteStrings("<html lang=\"", he.lang, "\">")
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
func (he *Encoder) WriteMeta(w io.Writer, m *meta.Meta, evalMeta encoder.EvalMetaFunc) (int, error) {
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

func (he *Encoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return he.WriteBlocks(w, &zn.Ast)
}

// WriteBlocks encodes a block slice.
func (he *Encoder) WriteBlocks(w io.Writer, bs *ast.BlockSlice) (int, error) {
	v := newVisitor(he, w)
	ast.Walk(v, bs)
	v.writeEndnotes()
	length, err := v.b.Flush()
	return length, err
}

// WriteInlines writes an inline slice to the writer
func (he *Encoder) WriteInlines(w io.Writer, is *ast.InlineSlice) (int, error) {
	v := newVisitor(he, w)
	v.inInteractive = he.interactive
	ast.Walk(v, is)
	length, err := v.b.Flush()
	return length, err
}
