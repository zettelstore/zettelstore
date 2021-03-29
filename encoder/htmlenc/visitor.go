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
	"sort"
	"strconv"
	"strings"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/strfun"
)

// visitor writes the abstract syntax tree to an io.Writer.
type visitor struct {
	env           *encoder.Environment
	b             encoder.BufWriter
	visibleSpace  bool // Show space character in raw text
	inVerse       bool // In verse block
	inInteractive bool // Rendered interactive HTML code
	lang          langStack
}

func newVisitor(he *htmlEncoder, w io.Writer) *visitor {
	var lang string
	if he.env != nil {
		lang = he.env.Lang
	}
	return &visitor{
		env:  he.env,
		b:    encoder.NewBufWriter(w),
		lang: newLangStack(lang),
	}
}

var mapMetaKey = map[string]string{
	meta.KeyCopyright: "copyright",
	meta.KeyLicense:   "license",
}

func (v *visitor) acceptMeta(m *meta.Meta) {
	for _, pair := range m.Pairs(true) {
		if env := v.env; env != nil && env.IgnoreMeta[pair.Key] {
			continue
		}
		if pair.Key == meta.KeyTitle {
			continue
		}
		if pair.Key == meta.KeyTags {
			v.writeTags(pair.Value)
		} else if key, ok := mapMetaKey[pair.Key]; ok {
			v.writeMeta("", key, pair.Value)
		} else {
			v.writeMeta("zs-", pair.Key, pair.Value)
		}
	}
}

func (v *visitor) writeTags(tags string) {
	v.b.WriteString("\n<meta name=\"keywords\" content=\"")
	for i, val := range meta.ListFromValue(tags) {
		if i > 0 {
			v.b.WriteString(", ")
		}
		v.writeQuotedEscaped(strings.TrimPrefix(val, "#"))
	}
	v.b.WriteString("\">")
}

func (v *visitor) writeMeta(prefix, key, value string) {
	v.b.WriteStrings("\n<meta name=\"", prefix, key, "\" content=\"")
	v.writeQuotedEscaped(value)
	v.b.WriteString("\">")
}

func (v *visitor) acceptBlockSlice(bns ast.BlockSlice) {
	for _, bn := range bns {
		bn.Accept(v)
	}
}
func (v *visitor) acceptItemSlice(ins ast.ItemSlice) {
	for _, in := range ins {
		in.Accept(v)
	}
}
func (v *visitor) acceptInlineSlice(ins ast.InlineSlice) {
	for _, in := range ins {
		in.Accept(v)
	}
}

func (v *visitor) writeEndnotes() {
	footnotes := v.env.GetCleanFootnotes()
	if len(footnotes) > 0 {
		v.b.WriteString("<ol class=\"zs-endnotes\">\n")
		for i := 0; i < len(footnotes); i++ {
			// Do not use a range loop above, because a footnote may contain
			// a footnote. Therefore v.enc.footnote may grow during the loop.
			fn := footnotes[i]
			n := strconv.Itoa(i + 1)
			v.b.WriteStrings("<li id=\"fn:", n, "\" role=\"doc-endnote\">")
			v.acceptInlineSlice(fn.Inlines)
			v.b.WriteStrings(
				" <a href=\"#fnref:",
				n,
				"\" class=\"zs-footnote-backref\" role=\"doc-backlink\">&#x21a9;&#xfe0e;</a></li>\n")
		}
		v.b.WriteString("</ol>\n")
	}
}

// visitAttributes write HTML attributes
func (v *visitor) visitAttributes(a *ast.Attributes) {
	if a == nil || len(a.Attrs) == 0 {
		return
	}
	keys := make([]string, 0, len(a.Attrs))
	for k := range a.Attrs {
		if k != "-" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	for _, k := range keys {
		if k == "" || k == "-" {
			continue
		}
		v.b.WriteStrings(" ", k)
		vl := a.Attrs[k]
		if len(vl) > 0 {
			v.b.WriteString("=\"")
			v.writeQuotedEscaped(vl)
			v.b.WriteByte('"')
		}
	}
}

func (v *visitor) writeHTMLEscaped(s string) {
	strfun.HTMLEscape(&v.b, s, v.visibleSpace)
}

func (v *visitor) writeQuotedEscaped(s string) {
	strfun.HTMLAttrEscape(&v.b, s)
}

func (v *visitor) writeReference(ref *ast.Reference) {
	if ref.URL == nil {
		v.writeHTMLEscaped(ref.Value)
		return
	}
	v.b.WriteString(ref.URL.String())
}
