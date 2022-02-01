//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package zettelmark provides a parser for zettelmarkup.
package zettelmark

import (
	"bytes"
	"fmt"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/input"
)

// parseInlineList parses a sequence of Inlines until EOS.
func (cp *zmkP) parseInlineList() *ast.InlineListNode {
	inp := cp.inp
	var ins []ast.InlineNode
	for inp.Ch != input.EOS {
		in := cp.parseInline()
		if in == nil {
			break
		}
		ins = append(ins, in)
	}
	return ast.CreateInlineListNode(ins...)
}

func (cp *zmkP) parseInline() ast.InlineNode {
	inp := cp.inp
	pos := inp.Pos
	if cp.nestingLevel <= maxNestingLevel {
		cp.nestingLevel++
		defer func() { cp.nestingLevel-- }()

		var in ast.InlineNode
		success := false
		switch inp.Ch {
		case input.EOS:
			return nil
		case '\n', '\r':
			return cp.parseSoftBreak()
		case ' ', '\t':
			return cp.parseSpace()
		case '[':
			inp.Next()
			switch inp.Ch {
			case '[':
				in, success = cp.parseLink()
			case '@':
				in, success = cp.parseCite()
			case '^':
				in, success = cp.parseFootnote()
			case '!':
				in, success = cp.parseMark()
			}
		case '{':
			inp.Next()
			if inp.Ch == '{' {
				in, success = cp.parseEmbed()
			}
		case '#':
			return cp.parseTag()
		case '%':
			in, success = cp.parseComment()
		case '_', '*', '>', '~', '\'', '^', ',', '<', '"', ':':
			in, success = cp.parseFormat()
		case '@', '+', '`', '=', runeModGrave:
			in, success = cp.parseLiteral()
		case '\\':
			return cp.parseBackslash()
		case '-':
			in, success = cp.parseNdash()
		case '&':
			in, success = cp.parseEntity()
		}
		if success {
			return in
		}
	}
	inp.SetPos(pos)
	return cp.parseText()
}

func (cp *zmkP) parseText() *ast.TextNode {
	inp := cp.inp
	pos := inp.Pos
	if inp.Ch == '\\' {
		return cp.parseTextBackslash()
	}
	for {
		inp.Next()
		switch inp.Ch {
		// The following case must contain all runes that occur in parseInline!
		// Plus the closing brackets ] and } and ) and the middle |
		case input.EOS, '\n', '\r', ' ', '\t', '[', ']', '{', '}', '(', ')', '|', '#', '%', '_', '*', '>', '~', '\'', '^', ',', '<', '"', ':', '+', '@', '`', runeModGrave, '=', '\\', '-', '&':
			return &ast.TextNode{Text: string(inp.Src[pos:inp.Pos])}
		}
	}
}

func (cp *zmkP) parseTextBackslash() *ast.TextNode {
	cp.inp.Next()
	return cp.parseBackslashRest()
}

func (cp *zmkP) parseBackslash() ast.InlineNode {
	inp := cp.inp
	inp.Next()
	switch inp.Ch {
	case '\n', '\r':
		inp.EatEOL()
		return &ast.BreakNode{Hard: true}
	default:
		return cp.parseBackslashRest()
	}
}

func (cp *zmkP) parseBackslashRest() *ast.TextNode {
	inp := cp.inp
	if input.IsEOLEOS(inp.Ch) {
		return &ast.TextNode{Text: "\\"}
	}
	if inp.Ch == ' ' {
		inp.Next()
		return &ast.TextNode{Text: "\u00a0"}
	}
	pos := inp.Pos
	inp.Next()
	return &ast.TextNode{Text: string(inp.Src[pos:inp.Pos])}
}

func (cp *zmkP) parseSpace() *ast.SpaceNode {
	inp := cp.inp
	pos := inp.Pos
	for {
		inp.Next()
		switch inp.Ch {
		case ' ', '\t':
		default:
			return &ast.SpaceNode{Lexeme: string(inp.Src[pos:inp.Pos])}
		}
	}
}

func (cp *zmkP) parseSoftBreak() *ast.BreakNode {
	cp.inp.EatEOL()
	return &ast.BreakNode{}
}

func (cp *zmkP) parseLink() (*ast.LinkNode, bool) {
	if ref, iln, ok := cp.parseReference(']'); ok {
		attrs := cp.parseAttributes(false)
		if len(ref) > 0 {
			onlyRef := false
			r := ast.ParseReference(ref)
			if iln == nil {
				iln = ast.CreateInlineListNode(&ast.TextNode{Text: ref})
				onlyRef = true
			}
			return &ast.LinkNode{
				Ref:     r,
				Inlines: iln,
				OnlyRef: onlyRef,
				Attrs:   attrs,
			}, true
		}
	}
	return nil, false
}

func (cp *zmkP) parseReference(closeCh rune) (ref string, iln *ast.InlineListNode, ok bool) {
	inp := cp.inp
	inp.Next()
	cp.skipSpace()
	pos := inp.Pos
	hasSpace, ok := cp.readReferenceToSep(closeCh)
	if !ok {
		return "", nil, false
	}
	var ins []ast.InlineNode
	if inp.Ch == '|' { // First part must be inline text
		if pos == inp.Pos { // [[| or {{|
			return "", nil, false
		}
		cp.inp = input.NewInput(inp.Src[pos:inp.Pos])
		for {
			in := cp.parseInline()
			if in == nil {
				break
			}
			ins = append(ins, in)
		}
		cp.inp = inp
		inp.Next()
	} else if hasSpace {
		return "", nil, false
	} else {
		inp.SetPos(pos)
	}

	cp.skipSpace()
	pos = inp.Pos
	if !cp.readReferenceToClose(closeCh) {
		return "", nil, false
	}
	ref = string(inp.Src[pos:inp.Pos])
	inp.Next()
	if inp.Ch != closeCh {
		return "", nil, false
	}
	inp.Next()
	if len(ins) == 0 {
		return ref, nil, true
	}
	return ref, ast.CreateInlineListNode(ins...), true
}

func (cp *zmkP) readReferenceToSep(closeCh rune) (bool, bool) {
	hasSpace := false
	inp := cp.inp
	for {
		switch inp.Ch {
		case input.EOS:
			return false, false
		case '\n', '\r', ' ':
			hasSpace = true
		case '|':
			return hasSpace, true
		case '\\':
			inp.Next()
			switch inp.Ch {
			case input.EOS:
				return false, false
			case '\n', '\r':
				hasSpace = true
			}
		case '%':
			inp.Next()
			if inp.Ch == '%' {
				inp.SkipToEOL()
			}
			continue
		case closeCh:
			inp.Next()
			if inp.Ch == closeCh {
				return hasSpace, true
			}
			continue
		}
		inp.Next()
	}
}

func (cp *zmkP) readReferenceToClose(closeCh rune) bool {
	inp := cp.inp
	for {
		switch inp.Ch {
		case input.EOS, '\n', '\r', ' ':
			return false
		case '\\':
			inp.Next()
			switch inp.Ch {
			case input.EOS, '\n', '\r':
				return false
			}
		case closeCh:
			return true
		}
		inp.Next()
	}
}

func (cp *zmkP) parseCite() (*ast.CiteNode, bool) {
	inp := cp.inp
	inp.Next()
	switch inp.Ch {
	case ' ', ',', '|', ']', '\n', '\r':
		return nil, false
	}
	pos := inp.Pos
loop:
	for {
		switch inp.Ch {
		case input.EOS:
			return nil, false
		case ' ', ',', '|', ']', '\n', '\r':
			break loop
		}
		inp.Next()
	}
	posL := inp.Pos
	switch inp.Ch {
	case ' ', ',', '|':
		inp.Next()
	}
	ins, ok := cp.parseLinkLikeRest()
	if !ok {
		return nil, false
	}
	attrs := cp.parseAttributes(false)
	return &ast.CiteNode{Key: string(inp.Src[pos:posL]), Inlines: ins, Attrs: attrs}, true
}

func (cp *zmkP) parseFootnote() (*ast.FootnoteNode, bool) {
	cp.inp.Next()
	iln, ok := cp.parseLinkLikeRest()
	if !ok {
		return nil, false
	}
	attrs := cp.parseAttributes(false)
	if iln == nil {
		iln = &ast.InlineListNode{}
	}
	return &ast.FootnoteNode{Inlines: iln, Attrs: attrs}, true
}

func (cp *zmkP) parseLinkLikeRest() (*ast.InlineListNode, bool) {
	cp.skipSpace()
	var ins []ast.InlineNode
	inp := cp.inp
	for inp.Ch != ']' {
		in := cp.parseInline()
		if in == nil {
			return nil, false
		}
		ins = append(ins, in)
		if _, ok := in.(*ast.BreakNode); ok && input.IsEOLEOS(inp.Ch) {
			return nil, false
		}
	}
	inp.Next()
	if len(ins) == 0 {
		return nil, true
	}
	return ast.CreateInlineListNode(ins...), true
}

func (cp *zmkP) parseEmbed() (ast.InlineNode, bool) {
	if ref, iln, ok := cp.parseReference('}'); ok {
		attrs := cp.parseAttributes(false)
		if len(ref) > 0 {
			r := ast.ParseReference(ref)
			return &ast.EmbedRefNode{
				Ref:     r,
				Inlines: iln,
				Attrs:   attrs,
			}, true
		}
	}
	return nil, false
}

func (cp *zmkP) parseMark() (*ast.MarkNode, bool) {
	inp := cp.inp
	inp.Next()
	pos := inp.Pos
	for inp.Ch != ']' {
		if !isNameRune(inp.Ch) {
			return nil, false
		}
		inp.Next()
	}
	mn := &ast.MarkNode{Text: string(inp.Src[pos:inp.Pos])}
	inp.Next()
	return mn, true
}

func (cp *zmkP) parseTag() ast.InlineNode {
	inp := cp.inp
	posH := inp.Pos
	inp.Next()
	pos := inp.Pos
	for isNameRune(inp.Ch) {
		inp.Next()
	}
	if pos == inp.Pos || inp.Ch == '#' {
		return &ast.TextNode{Text: string(inp.Src[posH:inp.Pos])}
	}
	return &ast.TagNode{Tag: string(inp.Src[pos:inp.Pos])}
}

func (cp *zmkP) parseComment() (res *ast.LiteralNode, success bool) {
	inp := cp.inp
	inp.Next()
	if inp.Ch != '%' {
		return nil, false
	}
	for inp.Ch == '%' {
		inp.Next()
	}
	cp.skipSpace()
	pos := inp.Pos
	for {
		if input.IsEOLEOS(inp.Ch) {
			return &ast.LiteralNode{
				Kind:    ast.LiteralComment,
				Content: append([]byte(nil), inp.Src[pos:inp.Pos]...),
			}, true
		}
		inp.Next()
	}
}

var mapRuneFormat = map[rune]ast.FormatKind{
	'_':  ast.FormatEmph,
	'*':  ast.FormatStrong,
	'>':  ast.FormatInsert,
	'~':  ast.FormatDelete,
	'\'': ast.FormatMonospace,
	'^':  ast.FormatSuper,
	',':  ast.FormatSub,
	'<':  ast.FormatQuotation,
	'"':  ast.FormatQuote,
	':':  ast.FormatSpan,
}

func (cp *zmkP) parseFormat() (res ast.InlineNode, success bool) {
	inp := cp.inp
	fch := inp.Ch
	kind, ok := mapRuneFormat[fch]
	if !ok {
		panic(fmt.Sprintf("%q is not a formatting char", fch))
	}
	inp.Next() // read 2nd formatting character
	if inp.Ch != fch {
		return nil, false
	}
	inp.Next()
	fn := &ast.FormatNode{Kind: kind, Inlines: &ast.InlineListNode{}}
	for {
		if inp.Ch == input.EOS {
			return nil, false
		}
		if inp.Ch == fch {
			inp.Next()
			if inp.Ch == fch {
				inp.Next()
				fn.Attrs = cp.parseAttributes(false)
				return fn, true
			}
			fn.Inlines.Append(&ast.TextNode{Text: string(fch)})
		} else if in := cp.parseInline(); in != nil {
			if _, ok = in.(*ast.BreakNode); ok && input.IsEOLEOS(inp.Ch) {
				return nil, false
			}
			fn.Inlines.Append(in)
		}
	}
}

var mapRuneLiteral = map[rune]ast.LiteralKind{
	'@':          ast.LiteralZettel,
	'`':          ast.LiteralProg,
	runeModGrave: ast.LiteralProg,
	'+':          ast.LiteralKeyb,
	'=':          ast.LiteralOutput,
}

func (cp *zmkP) parseLiteral() (res ast.InlineNode, success bool) {
	inp := cp.inp
	fch := inp.Ch
	kind, ok := mapRuneLiteral[fch]
	if !ok {
		panic(fmt.Sprintf("%q is not a formatting char", fch))
	}
	inp.Next() // read 2nd formatting character
	if inp.Ch != fch {
		return nil, false
	}
	fn := &ast.LiteralNode{Kind: kind}
	inp.Next()
	var buf bytes.Buffer
	for {
		if inp.Ch == input.EOS {
			return nil, false
		}
		if inp.Ch == fch {
			if inp.Peek() == fch {
				inp.Next()
				inp.Next()
				fn.Attrs = cp.parseAttributes(false)
				fn.Content = buf.Bytes()
				return fn, true
			}
			buf.WriteRune(fch)
			inp.Next()
		} else {
			tn := cp.parseText()
			buf.WriteString(tn.Text)
		}
	}
}

func (cp *zmkP) parseNdash() (res *ast.TextNode, success bool) {
	inp := cp.inp
	if inp.Peek() != inp.Ch {
		return nil, false
	}
	inp.Next()
	inp.Next()
	return &ast.TextNode{Text: "\u2013"}, true
}

func (cp *zmkP) parseEntity() (res *ast.TextNode, success bool) {
	if text, ok := cp.inp.ScanEntity(); ok {
		return &ast.TextNode{Text: text}, true
	}
	return nil, false
}
