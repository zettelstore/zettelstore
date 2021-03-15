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
	"fmt"
	"strconv"
	"strings"

	"zettelstore.de/z/ast"
)

// VisitText writes text content.
func (v *visitor) VisitText(tn *ast.TextNode) {
	v.writeHTMLEscaped(tn.Text)
}

// VisitTag writes tag content.
func (v *visitor) VisitTag(tn *ast.TagNode) {
	// TODO: erst mal als span. Link wäre gut, muss man vermutlich via Callback lösen.
	v.b.WriteString("<span class=\"zettel-tag\">#")
	v.writeHTMLEscaped(tn.Tag)
	v.b.WriteString("</span>")
}

// VisitSpace emits a white space.
func (v *visitor) VisitSpace(sn *ast.SpaceNode) {
	if v.inVerse || v.env.IsXHTML() {
		v.b.WriteString(sn.Lexeme)
	} else {
		v.b.WriteByte(' ')
	}
}

// VisitBreak writes HTML code for line breaks.
func (v *visitor) VisitBreak(bn *ast.BreakNode) {
	if bn.Hard {
		if v.env.IsXHTML() {
			v.b.WriteString("<br />\n")
		} else {
			v.b.WriteString("<br>\n")
		}
	} else {
		v.b.WriteByte('\n')
	}
}

// VisitLink writes HTML code for links.
func (v *visitor) VisitLink(ln *ast.LinkNode) {
	ln, n := v.env.AdaptLink(ln)
	if n != nil {
		n.Accept(v)
		return
	}
	v.lang.push(ln.Attrs)
	defer v.lang.pop()

	switch ln.Ref.State {
	case ast.RefStateSelf, ast.RefStateFound, ast.RefStateHosted, ast.RefStateBased:
		v.writeAHref(ln.Ref, ln.Attrs, ln.Inlines)
	case ast.RefStateBroken:
		attrs := ln.Attrs.Clone()
		attrs = attrs.Set("class", "zs-broken")
		attrs = attrs.Set("title", "Zettel not found") // l10n
		v.writeAHref(ln.Ref, attrs, ln.Inlines)
	case ast.RefStateExternal:
		attrs := ln.Attrs.Clone()
		attrs = attrs.Set("class", "zs-external")
		if v.env.HasNewWindow() {
			attrs = attrs.Set("target", "_blank").Set("rel", "noopener noreferrer")
		}
		v.writeAHref(ln.Ref, attrs, ln.Inlines)
		if v.env != nil {
			v.b.WriteString(v.env.MarkerExternal)
		}
	default:
		if v.env.IsInteractive(v.inInteractive) {
			v.writeSpan(ln.Inlines, ln.Attrs)
			return
		}
		v.b.WriteString("<a href=\"")
		v.writeQuotedEscaped(ln.Ref.Value)
		v.b.WriteByte('"')
		v.visitAttributes(ln.Attrs)
		v.b.WriteByte('>')
		v.inInteractive = true
		v.acceptInlineSlice(ln.Inlines)
		v.inInteractive = false
		v.b.WriteString("</a>")
	}
}

func (v *visitor) writeAHref(ref *ast.Reference, attrs *ast.Attributes, ins ast.InlineSlice) {
	if v.env.IsInteractive(v.inInteractive) {
		v.writeSpan(ins, attrs)
		return
	}
	v.b.WriteString("<a href=\"")
	v.writeReference(ref)
	v.b.WriteByte('"')
	v.visitAttributes(attrs)
	v.b.WriteByte('>')
	v.inInteractive = true
	v.acceptInlineSlice(ins)
	v.inInteractive = false
	v.b.WriteString("</a>")
}

// VisitImage writes HTML code for images.
func (v *visitor) VisitImage(in *ast.ImageNode) {
	in, n := v.env.AdaptImage(in)
	if n != nil {
		n.Accept(v)
		return
	}
	v.lang.push(in.Attrs)
	defer v.lang.pop()

	if in.Ref == nil {
		v.b.WriteString("<img src=\"data:image/")
		switch in.Syntax {
		case "svg":
			v.b.WriteString("svg+xml;utf8,")
			v.writeQuotedEscaped(string(in.Blob))
		default:
			v.b.WriteStrings(in.Syntax, ";base64,")
			v.b.WriteBase64(in.Blob)
		}
	} else {
		v.b.WriteString("<img src=\"")
		v.writeReference(in.Ref)
	}
	v.b.WriteString("\" alt=\"")
	v.acceptInlineSlice(in.Inlines)
	v.b.WriteByte('"')
	v.visitAttributes(in.Attrs)
	if v.env.IsXHTML() {
		v.b.WriteString(" />")
	} else {
		v.b.WriteByte('>')
	}
}

// VisitCite writes code for citations.
func (v *visitor) VisitCite(cn *ast.CiteNode) {
	cn, n := v.env.AdaptCite(cn)
	if n != nil {
		n.Accept(v)
		return
	}
	if cn == nil {
		return
	}
	v.lang.push(cn.Attrs)
	defer v.lang.pop()
	v.b.WriteString(cn.Key)
	if len(cn.Inlines) > 0 {
		v.b.WriteString(", ")
		v.acceptInlineSlice(cn.Inlines)
	}
}

// VisitFootnote write HTML code for a footnote.
func (v *visitor) VisitFootnote(fn *ast.FootnoteNode) {
	v.lang.push(fn.Attrs)
	defer v.lang.pop()
	if v.env.IsInteractive(v.inInteractive) {
		return
	}

	v.footnotes = append(v.footnotes, fn)
	n := strconv.Itoa(len(v.footnotes))
	v.b.WriteStrings("<sup id=\"fnref:", n, "\"><a href=\"#fn:", n, "\" class=\"zs-footnote-ref\" role=\"doc-noteref\">", n, "</a></sup>")
	// TODO: what to do with Attrs?
}

// VisitMark writes HTML code to mark a position.
func (v *visitor) VisitMark(mn *ast.MarkNode) {
	if v.env.IsInteractive(v.inInteractive) {
		return
	}
	if len(mn.Text) > 0 {
		v.b.WriteStrings("<a id=\"", mn.Text, "\"></a>")
	}
}

// VisitFormat write HTML code for formatting text.
func (v *visitor) VisitFormat(fn *ast.FormatNode) {
	v.lang.push(fn.Attrs)
	defer v.lang.pop()

	var code string
	attrs := fn.Attrs
	switch fn.Code {
	case ast.FormatItalic:
		code = "i"
	case ast.FormatEmph:
		code = "em"
	case ast.FormatBold:
		code = "b"
	case ast.FormatStrong:
		code = "strong"
	case ast.FormatUnder:
		code = "u" // TODO: ändern in <span class="XXX">
	case ast.FormatInsert:
		code = "ins"
	case ast.FormatStrike:
		code = "s"
	case ast.FormatDelete:
		code = "del"
	case ast.FormatSuper:
		code = "sup"
	case ast.FormatSub:
		code = "sub"
	case ast.FormatQuotation:
		code = "q"
	case ast.FormatSmall:
		code = "small"
	case ast.FormatSpan:
		v.writeSpan(fn.Inlines, processSpanAttributes(attrs))
		return
	case ast.FormatMonospace:
		code = "span"
		attrs = attrs.Set("style", "font-family:monospace")
	case ast.FormatQuote:
		v.visitQuotes(fn)
		return
	default:
		panic(fmt.Sprintf("Unknown format code %v", fn.Code))
	}
	v.b.WriteStrings("<", code)
	v.visitAttributes(attrs)
	v.b.WriteByte('>')
	v.acceptInlineSlice(fn.Inlines)
	v.b.WriteStrings("</", code, ">")
}

func (v *visitor) writeSpan(ins ast.InlineSlice, attrs *ast.Attributes) {
	v.b.WriteString("<span")
	v.visitAttributes(attrs)
	v.b.WriteByte('>')
	v.acceptInlineSlice(ins)
	v.b.WriteString("</span>")

}

var langQuotes = map[string][2]string{
	"en": {"&ldquo;", "&rdquo;"},
	"de": {"&bdquo;", "&ldquo;"},
	"fr": {"&laquo;&nbsp;", "&nbsp;&raquo;"},
}

func getQuotes(lang string) (string, string) {
	langFields := strings.FieldsFunc(lang, func(r rune) bool { return r == '-' || r == '_' })
	for len(langFields) > 0 {
		langSup := strings.Join(langFields, "-")
		quotes, ok := langQuotes[langSup]
		if ok {
			return quotes[0], quotes[1]
		}
		langFields = langFields[0 : len(langFields)-1]
	}
	return "\"", "\""
}

func (v *visitor) visitQuotes(fn *ast.FormatNode) {
	_, withSpan := fn.Attrs.Get("lang")
	if withSpan {
		v.b.WriteString("<span")
		v.visitAttributes(fn.Attrs)
		v.b.WriteByte('>')
	}
	openingQ, closingQ := getQuotes(v.lang.top())
	v.b.WriteString(openingQ)
	v.acceptInlineSlice(fn.Inlines)
	v.b.WriteString(closingQ)
	if withSpan {
		v.b.WriteString("</span>")
	}
}

// VisitLiteral write HTML code for literal inline text.
func (v *visitor) VisitLiteral(ln *ast.LiteralNode) {
	switch ln.Code {
	case ast.LiteralProg:
		v.writeLiteral("<code", "</code>", ln.Attrs, ln.Text)
	case ast.LiteralKeyb:
		v.writeLiteral("<kbd", "</kbd>", ln.Attrs, ln.Text)
	case ast.LiteralOutput:
		v.writeLiteral("<samp", "</samp>", ln.Attrs, ln.Text)
	case ast.LiteralComment:
		v.b.WriteString("<!-- ")
		v.writeHTMLEscaped(ln.Text) // writeCommentEscaped
		v.b.WriteString(" -->")
	case ast.LiteralHTML:
		v.b.WriteString(ln.Text)
	default:
		panic(fmt.Sprintf("Unknown literal code %v", ln.Code))
	}
}

func (v *visitor) writeLiteral(codeS, codeE string, attrs *ast.Attributes, text string) {
	oldVisible := v.visibleSpace
	if attrs != nil {
		v.visibleSpace = attrs.HasDefault()
	}
	v.b.WriteString(codeS)
	v.visitAttributes(attrs)
	v.b.WriteByte('>')
	v.writeHTMLEscaped(text)
	v.b.WriteString(codeE)
	v.visibleSpace = oldVisible
}
