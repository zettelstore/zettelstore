//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
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

	"zettelstore.de/c/api"
	"zettelstore.de/c/html"
	"zettelstore.de/c/zjson"
	"zettelstore.de/z/ast"
)

func (v *visitor) visitBreak(bn *ast.BreakNode) {
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

func (v *visitor) visitLink(ln *ast.LinkNode) {
	switch ln.Ref.State {
	case ast.RefStateSelf, ast.RefStateFound, ast.RefStateHosted, ast.RefStateBased:
		v.writeAHref(ln.Ref, ln.Attrs, &ln.Inlines)
	case ast.RefStateBroken:
		attrs := ln.Attrs.Clone()
		attrs = attrs.Set("class", "broken")
		attrs = attrs.Set("title", "Zettel not found") // l10n
		v.writeAHref(ln.Ref, attrs, &ln.Inlines)
	case ast.RefStateExternal:
		attrs := ln.Attrs.Clone()
		attrs = attrs.Set("class", "external")
		if v.env.HasNewWindow() {
			attrs = attrs.Set("target", "_blank").Set("rel", "noopener noreferrer")
		}
		v.writeAHref(ln.Ref, attrs, &ln.Inlines)
		if v.env != nil {
			v.b.WriteString(v.env.MarkerExternal)
		}
	default:
		if v.env.IsInteractive(v.inInteractive) {
			v.writeSpan(&ln.Inlines, ln.Attrs)
			return
		}
		v.b.WriteString("<a href=\"")
		v.writeQuotedEscaped(ln.Ref.Value)
		v.b.WriteByte('"')
		v.visitAttributes(ln.Attrs)
		v.b.WriteByte('>')
		v.writeLinkInlines(&ln.Inlines, ln.Ref)
		v.inInteractive = false
		v.b.WriteString("</a>")
	}
}

func (v *visitor) writeAHref(ref *ast.Reference, attrs zjson.Attributes, is *ast.InlineSlice) {
	if v.env.IsInteractive(v.inInteractive) {
		v.writeSpan(is, attrs)
		return
	}
	v.b.WriteString("<a href=\"")
	v.writeReference(ref)
	v.b.WriteByte('"')
	v.visitAttributes(attrs)
	v.b.WriteByte('>')
	v.writeLinkInlines(is, ref)
	v.b.WriteString("</a>")
}
func (v *visitor) writeLinkInlines(is *ast.InlineSlice, ref *ast.Reference) {
	saveInteractive := v.inInteractive
	v.inInteractive = true
	if len(*is) == 0 {
		v.writeHTMLEscaped(ref.Value)
	} else {
		ast.Walk(v, is)
	}
	v.inInteractive = saveInteractive
}

func (v *visitor) visitEmbedRef(en *ast.EmbedRefNode) {
	v.b.WriteString("<img src=\"")
	v.writeReference(en.Ref)
	v.b.WriteString("\" alt=\"")
	ast.Walk(v, &en.Inlines) // TODO: wrong, must use textenc
	v.b.WriteByte('"')
	v.visitAttributes(en.Attrs)
	if v.env.IsXHTML() {
		v.b.WriteString(" />")
	} else {
		v.b.WriteByte('>')
	}
}

func (v *visitor) visitEmbedBLOB(en *ast.EmbedBLOBNode) {
	if en.Syntax == api.ValueSyntaxSVG {
		v.b.Write(en.Blob)
		return
	}

	v.b.WriteString("<img src=\"data:image/")
	v.b.WriteStrings(en.Syntax, ";base64,")
	v.b.WriteBase64(en.Blob)
	v.b.WriteString("\" alt=\"")
	ast.Walk(v, &en.Inlines)
	v.b.WriteByte('"')
	v.visitAttributes(en.Attrs)
	if v.env.IsXHTML() {
		v.b.WriteString(" />")
	} else {
		v.b.WriteByte('>')
	}
}

func (v *visitor) visitCite(cn *ast.CiteNode) {
	v.b.WriteString(cn.Key)
	if len(cn.Inlines) > 0 {
		v.b.WriteString(", ")
		ast.Walk(v, &cn.Inlines)
	}
}

func (v *visitor) visitFootnote(fn *ast.FootnoteNode) {
	if v.env.IsInteractive(v.inInteractive) {
		return
	}

	n := strconv.Itoa(v.env.AddFootnote(fn))
	v.b.WriteStrings("<sup id=\"fnref:", n, "\"><a href=\"#fn:", n, "\" class=\"footnote-ref\" role=\"doc-noteref\">", n, "</a></sup>")
	// TODO: what to do with Attrs?
}

func (v *visitor) visitMark(mn *ast.MarkNode) {
	if v.env.IsInteractive(v.inInteractive) {
		return
	}
	if fragment := mn.Fragment; fragment != "" {
		v.b.WriteStrings(`<a id="`, fragment, `">`)
		if len(mn.Inlines) > 0 {
			ast.Walk(v, &mn.Inlines)
		}
		v.b.WriteString("</a>")
	}
}

func (v *visitor) visitFormat(fn *ast.FormatNode) {
	var code string
	switch fn.Kind {
	case ast.FormatEmph:
		code = "em"
	case ast.FormatStrong:
		code = "strong"
	case ast.FormatInsert:
		code = "ins"
	case ast.FormatDelete:
		code = "del"
	case ast.FormatSuper:
		code = "sup"
	case ast.FormatSub:
		code = "sub"
	case ast.FormatQuote:
		code = "q"
	case ast.FormatSpan:
		v.writeSpan(&fn.Inlines, processSpanAttributes(fn.Attrs))
		return
	default:
		panic(fmt.Sprintf("Unknown format kind %v", fn.Kind))
	}
	v.b.WriteStrings("<", code)
	v.visitAttributes(fn.Attrs)
	v.b.WriteByte('>')
	ast.Walk(v, &fn.Inlines)
	v.b.WriteStrings("</", code, ">")
}

func (v *visitor) writeSpan(is *ast.InlineSlice, attrs zjson.Attributes) {
	v.b.WriteString("<span")
	v.visitAttributes(attrs)
	v.b.WriteByte('>')
	ast.Walk(v, is)
	v.b.WriteString("</span>")

}

func (v *visitor) visitLiteral(ln *ast.LiteralNode) {
	switch ln.Kind {
	case ast.LiteralProg:
		v.writeLiteral("<code", "</code>", ln.Attrs, ln.Content)
	case ast.LiteralMath:
		v.writeLiteral("<code", "</code>", ln.Attrs.Clone().AddClass("zs-math"), ln.Content)
	case ast.LiteralInput:
		v.writeLiteral("<kbd", "</kbd>", ln.Attrs, ln.Content)
	case ast.LiteralOutput:
		v.writeLiteral("<samp", "</samp>", ln.Attrs, ln.Content)
	case ast.LiteralZettel, ast.LiteralComment:
		if v.inlinePos > 0 {
			v.b.WriteByte(' ')
		}
		v.b.WriteString("<!-- ")
		v.writeHTMLEscaped(string(ln.Content)) // writeCommentEscaped
		v.b.WriteString(" -->")
	case ast.LiteralHTML:
		if html.IsSave(string(ln.Content)) {
			v.b.Write(ln.Content)
		}
	default:
		panic(fmt.Sprintf("Unknown literal kind %v", ln.Kind))
	}
}

func (v *visitor) writeLiteral(codeS, codeE string, attrs zjson.Attributes, content []byte) {
	oldVisible := v.visibleSpace
	if attrs != nil {
		v.visibleSpace = attrs.HasDefault()
	}
	v.b.WriteString(codeS)
	v.visitAttributes(attrs)
	v.b.WriteByte('>')
	v.writeHTMLEscaped(string(content))
	v.b.WriteString(codeE)
	v.visibleSpace = oldVisible
}
