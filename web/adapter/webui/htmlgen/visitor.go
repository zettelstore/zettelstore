//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package htmlgen

import (
	"bytes"
	"strconv"
	"strings"

	"zettelstore.de/c/html"
	"zettelstore.de/c/zjson"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/encoder/textenc"
)

// visitor writes the abstract syntax tree to an io.Writer.
type visitor struct {
	he           *Encoder
	b            encoder.EncWriter
	visibleSpace bool // Show space character in plain text
	inVerse      bool // In verse block
	textEnc      encoder.Encoder
	inlinePos    int // Element position in inline list node
}

func newVisitor(he *Encoder, buf *bytes.Buffer) *visitor {
	return &visitor{
		he:      he,
		b:       encoder.NewEncWriter(buf),
		textEnc: textenc.Create(),
	}
}

func (v *visitor) makeResult(buf *bytes.Buffer) (string, error) {
	if _, err := v.b.Flush(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (v *visitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.BlockSlice:
		for i, bn := range *n {
			if i > 0 {
				v.b.WriteByte('\n')
			}
			ast.Walk(v, bn)
		}
	case *ast.InlineSlice:
		for i, in := range *n {
			v.inlinePos = i
			ast.Walk(v, in)
		}
		v.inlinePos = 0
	case *ast.ParaNode:
		v.b.WriteString("<p>")
		ast.Walk(v, &n.Inlines)
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
		v.b.WriteBytes('>')
	case *ast.NestedListNode:
		v.visitNestedList(n)
	case *ast.DescriptionListNode:
		v.visitDescriptionList(n)
	case *ast.TableNode:
		v.visitTable(n)
	case *ast.TranscludeNode:
		return nil // Nothing to write. Or: an iFrame?
	case *ast.BLOBNode:
		v.visitBLOB(n)
	case *ast.TextNode:
		v.writeHTMLEscaped(n.Text)
	case *ast.TagNode:
		v.b.WriteString("<span class=\"zettel-tag\">#")
		v.writeHTMLEscaped(n.Tag)
		v.b.WriteString("</span>")
	case *ast.SpaceNode:
		if v.inVerse {
			v.b.WriteString(n.Lexeme)
		} else {
			v.b.WriteByte(' ')
		}
	case *ast.BreakNode:
		v.visitBreak(n)
	case *ast.LinkNode:
		v.visitLink(n)
	case *ast.EmbedRefNode:
		v.visitEmbedRef(n)
	case *ast.EmbedBLOBNode:
		v.visitEmbedBLOB(n)
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

func (v *visitor) evalValue(value string, evalMeta encoder.EvalMetaFunc) string {
	var buf bytes.Buffer
	is := evalMeta(value)
	_, err := v.textEnc.WriteInlines(&buf, &is)
	if err == nil {
		return buf.String()
	}
	return ""
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
	fn, fnNum := v.he.popFootnote()
	if fn == nil {
		return
	}
	v.b.WriteString("\n<ol class=\"zs-endnotes\">\n")
	for fn != nil {
		n := strconv.Itoa(fnNum)
		v.b.WriteStrings(`<li class="zs-endnote" id="fn:`, n, `" role="doc-endnote" value="`, n, `">`)
		ast.Walk(v, &fn.Inlines)
		v.b.WriteStrings(
			" <a class=\"zs-endnote-backref\" href=\"#fnref:",
			n,
			"\" role=\"doc-backlink\">&#x21a9;&#xfe0e;</a></li>\n")
		fn, fnNum = v.he.popFootnote()
	}
	v.b.WriteString("</ol>\n")
}

// visitAttributes write HTML attributes
func (v *visitor) visitAttributes(a zjson.Attributes) {
	if a.IsEmpty() {
		return
	}
	a = a.Clone().RemoveDefault()

	for _, k := range a.Keys() {
		if k == "" || k == "-" {
			continue
		}
		v.b.WriteStrings(" ", k)
		vl := a[k]
		if len(vl) > 0 {
			v.b.WriteString("=\"")
			v.writeQuotedEscaped(vl)
			v.b.WriteByte('"')
		}
	}
}

func (v *visitor) writeHTMLEscaped(s string) { html.Escape(&v.b, s) }
func (v *visitor) writeHTMLLiteralEscaped(s string) {
	if v.visibleSpace {
		html.EscapeVisible(&v.b, s)
	} else {
		html.EscapeLiteral(&v.b, s)
	}
}

func (v *visitor) writeQuotedEscaped(s string) {
	html.AttributeEscape(&v.b, s)
}

func (v *visitor) writeReference(ref *ast.Reference) {
	if ref.URL == nil {
		v.writeHTMLEscaped(ref.Value)
		return
	}
	v.b.WriteString(ref.URL.String())
}
