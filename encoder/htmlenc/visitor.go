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

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/strfun"
)

// visitor writes the abstract syntax tree to an io.Writer.
type visitor struct {
	env           *encoder.Environment
	b             encoder.BufWriter
	visibleSpace  bool // Show space character in plain text
	inVerse       bool // In verse block
	inInteractive bool // Rendered interactive HTML code
	lang          langStack
	textEnc       encoder.Encoder
}

func newVisitor(he *htmlEncoder, w io.Writer) *visitor {
	var lang string
	if he.env != nil {
		lang = he.env.Lang
	}
	return &visitor{
		env:     he.env,
		b:       encoder.NewBufWriter(w),
		lang:    newLangStack(lang),
		textEnc: encoder.Create(api.EncoderText, nil),
	}
}

func (v *visitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.BlockListNode:
		for i, bn := range n.List {
			if i > 0 {
				v.b.WriteByte('\n')
			}
			ast.Walk(v, bn)
		}
	case *ast.ParaNode:
		v.b.WriteString("<p>")
		ast.Walk(v, n.Inlines)
		v.writeEndPara()
	case *ast.VerbatimNode:
		v.visitVerbatim(n)
	case *ast.RegionNode:
		v.visitRegion(n)
	case *ast.HeadingNode:
		v.visitHeading(n)
	case *ast.HRuleNode:
		v.b.WriteString("<hr")
		v.visitAttributes(n.Attrs)
		if v.env.IsXHTML() {
			v.b.WriteString(" />")
		} else {
			v.b.WriteBytes('>')
		}
	case *ast.NestedListNode:
		v.visitNestedList(n)
	case *ast.DescriptionListNode:
		v.visitDescriptionList(n)
	case *ast.TableNode:
		v.visitTable(n)
	case *ast.BLOBNode:
		v.visitBLOB(n)
	case *ast.TextNode:
		v.writeHTMLEscaped(n.Text)
	case *ast.TagNode:
		// TODO: erst mal als span. Link wäre gut, muss man vermutlich via Callback lösen.
		v.b.WriteString("<span class=\"zettel-tag\">#")
		v.writeHTMLEscaped(n.Tag)
		v.b.WriteString("</span>")
	case *ast.SpaceNode:
		if v.inVerse || v.env.IsXHTML() {
			v.b.WriteString(n.Lexeme)
		} else {
			v.b.WriteByte(' ')
		}
	case *ast.BreakNode:
		v.visitBreak(n)
	case *ast.LinkNode:
		v.visitLink(n)
	case *ast.EmbedNode:
		v.visitEmbed(n)
	case *ast.CiteNode:
		v.visitCite(n)
	case *ast.FootnoteNode:
		v.visitFootnote(n)
	case *ast.MarkNode:
		v.visitMark(n)
	case *ast.FormatNode:
		v.visitFormat(n)
	case *ast.LiteralNode:
		v.visitLiteral(n)
	default:
		return v
	}
	return nil
}

var mapMetaKey = map[string]string{
	api.KeyCopyright: "copyright",
	api.KeyLicense:   "license",
}

func (v *visitor) acceptMeta(m *meta.Meta, evalMeta encoder.EvalMetaFunc) {
	ignore := v.setupIgnoreSet()
	ignore[api.KeyTitle] = true
	if tags, ok := m.Get(api.KeyAllTags); ok {
		v.writeTags(tags)
		ignore[api.KeyAllTags] = true
		ignore[api.KeyTags] = true
	} else if tags, ok := m.Get(api.KeyTags); ok {
		v.writeTags(tags)
		ignore[api.KeyTags] = true
	}

	for _, p := range m.Pairs(true) {
		key := p.Key
		if ignore[key] {
			continue
		}
		value := p.Value
		if m.Type(key) == meta.TypeZettelmarkup {
			if v := v.evalValue(value, evalMeta); v != "" {
				value = v
			}
		}
		if mKey, ok := mapMetaKey[key]; ok {
			v.writeMeta("", mKey, value)
		} else {
			v.writeMeta("zs-", key, value)
		}
	}
}

func (v *visitor) evalValue(value string, evalMeta encoder.EvalMetaFunc) string {
	var sb strings.Builder
	_, err := v.textEnc.WriteInlines(&sb, evalMeta(value))
	if err == nil {
		return sb.String()
	}
	return ""
}

func (v *visitor) setupIgnoreSet() map[string]bool {
	if v.env == nil || v.env.IgnoreMeta == nil {
		return make(map[string]bool)
	}
	result := make(map[string]bool, len(v.env.IgnoreMeta))
	for k, v := range v.env.IgnoreMeta {
		if v {
			result[k] = true
		}
	}
	return result
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
			ast.Walk(v, fn.Inlines)
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
	if a.IsEmpty() {
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
