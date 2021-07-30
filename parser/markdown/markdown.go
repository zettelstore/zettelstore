//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package markdown provides a parser for markdown.
package markdown

import (
	"bytes"
	"fmt"
	"strings"

	gm "github.com/yuin/goldmark"
	gmAst "github.com/yuin/goldmark/ast"
	gmText "github.com/yuin/goldmark/text"

	"zettelstore.de/z/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
)

func init() {
	parser.Register(&parser.Info{
		Name:         "markdown",
		AltNames:     []string{"md"},
		ParseBlocks:  parseBlocks,
		ParseInlines: parseInlines,
	})
}

func parseBlocks(inp *input.Input, m *meta.Meta, syntax string) *ast.BlockListNode {
	p := parseMarkdown(inp)
	return &ast.BlockListNode{List: p.acceptBlockSlice(p.docNode)}
}

func parseInlines(inp *input.Input, syntax string) *ast.InlineListNode {
	panic("markdown.parseInline not yet implemented")
}

func parseMarkdown(inp *input.Input) *mdP {
	source := []byte(inp.Src[inp.Pos:])
	parser := gm.DefaultParser()
	node := parser.Parse(gmText.NewReader(source))
	textEnc := encoder.Create(api.EncoderText, nil)
	return &mdP{source: source, docNode: node, textEnc: textEnc}
}

type mdP struct {
	source  []byte
	docNode gmAst.Node
	textEnc encoder.Encoder
}

func (p *mdP) acceptBlockSlice(docNode gmAst.Node) ast.BlockSlice {
	if docNode.Type() != gmAst.TypeDocument {
		panic(fmt.Sprintf("Expected document, but got node type %v", docNode.Type()))
	}
	result := make(ast.BlockSlice, 0, docNode.ChildCount())
	for child := docNode.FirstChild(); child != nil; child = child.NextSibling() {
		if block := p.acceptBlock(child); block != nil {
			result = append(result, block)
		}
	}
	return result
}

func (p *mdP) acceptBlock(node gmAst.Node) ast.ItemNode {
	if node.Type() != gmAst.TypeBlock {
		panic(fmt.Sprintf("Expected block node, but got node type %v", node.Type()))
	}
	switch n := node.(type) {
	case *gmAst.Paragraph:
		return p.acceptParagraph(n)
	case *gmAst.TextBlock:
		return p.acceptTextBlock(n)
	case *gmAst.Heading:
		return p.acceptHeading(n)
	case *gmAst.ThematicBreak:
		return p.acceptThematicBreak(n)
	case *gmAst.CodeBlock:
		return p.acceptCodeBlock(n)
	case *gmAst.FencedCodeBlock:
		return p.acceptFencedCodeBlock(n)
	case *gmAst.Blockquote:
		return p.acceptBlockquote(n)
	case *gmAst.List:
		return p.acceptList(n)
	case *gmAst.HTMLBlock:
		return p.acceptHTMLBlock(n)
	}
	panic(fmt.Sprintf("Unhandled block node of kind %v", node.Kind()))
}

func (p *mdP) acceptParagraph(node *gmAst.Paragraph) ast.ItemNode {
	if iln := p.acceptChildren(node); iln != nil && len(iln.List) > 0 {
		return &ast.ParaNode{Inlines: iln}
	}
	return nil
}

func (p *mdP) acceptHeading(node *gmAst.Heading) *ast.HeadingNode {
	return &ast.HeadingNode{
		Level:   node.Level,
		Inlines: p.acceptChildren(node),
		Attrs:   nil,
	}
}

func (p *mdP) acceptThematicBreak(node *gmAst.ThematicBreak) *ast.HRuleNode {
	return &ast.HRuleNode{
		Attrs: nil, //TODO
	}
}

func (p *mdP) acceptCodeBlock(node *gmAst.CodeBlock) *ast.VerbatimNode {
	return &ast.VerbatimNode{
		Kind:  ast.VerbatimProg,
		Attrs: nil, //TODO
		Lines: p.acceptRawText(node),
	}
}

func (p *mdP) acceptFencedCodeBlock(node *gmAst.FencedCodeBlock) *ast.VerbatimNode {
	var attrs *ast.Attributes
	if language := node.Language(p.source); len(language) > 0 {
		attrs = attrs.Set("class", "language-"+cleanText(string(language), true))
	}
	return &ast.VerbatimNode{
		Kind:  ast.VerbatimProg,
		Attrs: attrs,
		Lines: p.acceptRawText(node),
	}
}

func (p *mdP) acceptRawText(node gmAst.Node) []string {
	lines := node.Lines()
	result := make([]string, 0, lines.Len())
	for i := 0; i < lines.Len(); i++ {
		s := lines.At(i)
		line := s.Value(p.source)
		if l := len(line); l > 0 {
			if l > 1 && line[l-2] == '\r' && line[l-1] == '\n' {
				line = line[0 : l-2]
			} else if line[l-1] == '\n' || line[l-1] == '\r' {
				line = line[0 : l-1]
			}
		}
		result = append(result, string(line))
	}
	return result
}

func (p *mdP) acceptBlockquote(node *gmAst.Blockquote) *ast.NestedListNode {
	return &ast.NestedListNode{
		Kind: ast.NestedListQuote,
		Items: []ast.ItemSlice{
			p.acceptItemSlice(node),
		},
	}
}

func (p *mdP) acceptList(node *gmAst.List) ast.ItemNode {
	kind := ast.NestedListUnordered
	var attrs *ast.Attributes
	if node.IsOrdered() {
		kind = ast.NestedListOrdered
		if node.Start != 1 {
			attrs = attrs.Set("start", fmt.Sprintf("%d", node.Start))
		}
	}
	items := make([]ast.ItemSlice, 0, node.ChildCount())
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		item, ok := child.(*gmAst.ListItem)
		if !ok {
			panic(fmt.Sprintf("Expected list item node, but got %v", child.Kind()))
		}
		items = append(items, p.acceptItemSlice(item))
	}
	return &ast.NestedListNode{
		Kind:  kind,
		Items: items,
		Attrs: attrs,
	}
}

func (p *mdP) acceptItemSlice(node gmAst.Node) ast.ItemSlice {
	result := make(ast.ItemSlice, 0, node.ChildCount())
	for elem := node.FirstChild(); elem != nil; elem = elem.NextSibling() {
		if item := p.acceptBlock(elem); item != nil {
			result = append(result, item)
		}
	}
	return result
}

func (p *mdP) acceptTextBlock(node *gmAst.TextBlock) ast.ItemNode {
	if iln := p.acceptChildren(node); iln != nil && len(iln.List) > 0 {
		return &ast.ParaNode{Inlines: iln}
	}
	return nil
}

func (p *mdP) acceptHTMLBlock(node *gmAst.HTMLBlock) *ast.VerbatimNode {
	lines := p.acceptRawText(node)
	if node.HasClosure() {
		closure := string(node.ClosureLine.Value(p.source))
		if l := len(closure); l > 1 && closure[l-1] == '\n' {
			closure = closure[:l-1]
		}
		lines = append(lines, closure)
	}
	return &ast.VerbatimNode{
		Kind:  ast.VerbatimHTML,
		Lines: lines,
	}
}

func (p *mdP) acceptChildren(node gmAst.Node) *ast.InlineListNode {
	result := make([]ast.InlineNode, 0, node.ChildCount())
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if inlines := p.acceptInline(child); inlines != nil {
			result = append(result, inlines...)
		}
	}
	return &ast.InlineListNode{List: result}
}

func (p *mdP) acceptInline(node gmAst.Node) ast.InlineSlice {
	if node.Type() != gmAst.TypeInline {
		panic(fmt.Sprintf("Expected inline node, but got %v", node.Type()))
	}
	switch n := node.(type) {
	case *gmAst.Text:
		return p.acceptText(n)
	case *gmAst.CodeSpan:
		return p.acceptCodeSpan(n)
	case *gmAst.Emphasis:
		return p.acceptEmphasis(n)
	case *gmAst.Link:
		return p.acceptLink(n)
	case *gmAst.Image:
		return p.acceptImage(n)
	case *gmAst.AutoLink:
		return p.acceptAutoLink(n)
	case *gmAst.RawHTML:
		return p.acceptRawHTML(n)
	}
	panic(fmt.Sprintf("Unhandled inline node %v", node.Kind()))
}

func (p *mdP) acceptText(node *gmAst.Text) ast.InlineSlice {
	segment := node.Segment
	if node.IsRaw() {
		return splitText(string(segment.Value(p.source)))
	}
	ins := splitText(string(segment.Value(p.source)))
	result := make(ast.InlineSlice, 0, len(ins)+1)
	for _, in := range ins {
		if tn, ok := in.(*ast.TextNode); ok {
			tn.Text = cleanText(tn.Text, true)
		}
		result = append(result, in)
	}
	if node.HardLineBreak() {
		result = append(result, &ast.BreakNode{Hard: true})
	} else if node.SoftLineBreak() {
		result = append(result, &ast.BreakNode{Hard: false})
	}
	return result
}

// splitText transform the text into a sequence of TextNode and SpaceNode
func splitText(text string) ast.InlineSlice {
	if text == "" {
		return ast.InlineSlice{}
	}
	result := make(ast.InlineSlice, 0, 1)

	state := 0 // 0=unknown,1=non-spaces,2=spaces
	lastPos := 0
	for pos, ch := range text {
		if input.IsSpace(ch) {
			if state == 1 {
				result = append(result, &ast.TextNode{Text: text[lastPos:pos]})
				lastPos = pos
			}
			state = 2
		} else {
			if state == 2 {
				result = append(result, &ast.SpaceNode{Lexeme: text[lastPos:pos]})
				lastPos = pos
			}
			state = 1
		}
	}
	switch state {
	case 1:
		result = append(result, &ast.TextNode{Text: text[lastPos:]})
	case 2:
		result = append(result, &ast.SpaceNode{Lexeme: text[lastPos:]})
	default:
		panic(fmt.Sprintf("Unexpected state %v", state))
	}
	return result
}

var ignoreAfterBS = map[byte]bool{
	'!': true, '"': true, '#': true, '$': true, '%': true, '&': true,
	'\'': true, '(': true, ')': true, '*': true, '+': true, ',': true,
	'-': true, '.': true, '/': true, ':': true, ';': true, '<': true,
	'=': true, '>': true, '?': true, '@': true, '[': true, '\\': true,
	']': true, '^': true, '_': true, '`': true, '{': true, '|': true,
	'}': true, '~': true,
}

// cleanText removes backslashes from TextNodes and expands entities
func cleanText(text string, cleanBS bool) string {
	lastPos := 0
	var sb strings.Builder
	for pos, ch := range text {
		if pos < lastPos {
			continue
		}
		if ch == '&' {
			inp := input.NewInput(text[pos:])
			if s, ok := inp.ScanEntity(); ok {
				sb.WriteString(text[lastPos:pos])
				sb.WriteString(s)
				lastPos = pos + inp.Pos
			}
			continue
		}
		if cleanBS && ch == '\\' && pos < len(text)-1 && ignoreAfterBS[text[pos+1]] {
			sb.WriteString(text[lastPos:pos])
			sb.WriteByte(text[pos+1])
			lastPos = pos + 2
		}
	}
	if lastPos == 0 {
		return text
	}
	if lastPos < len(text) {
		sb.WriteString(text[lastPos:])
	}
	return sb.String()
}

func (p *mdP) acceptCodeSpan(node *gmAst.CodeSpan) ast.InlineSlice {
	return ast.InlineSlice{
		&ast.LiteralNode{
			Kind:  ast.LiteralProg,
			Attrs: nil, //TODO
			Text:  cleanCodeSpan(string(node.Text(p.source))),
		},
	}
}

func cleanCodeSpan(text string) string {
	if text == "" {
		return ""
	}
	lastPos := 0
	var sb strings.Builder
	for pos, ch := range text {
		if ch == '\n' {
			sb.WriteString(text[lastPos:pos])
			if pos < len(text)-1 {
				sb.WriteByte(' ')
			}
			lastPos = pos + 1
		}
	}
	if lastPos == 0 {
		return text
	}
	sb.WriteString(text[lastPos:])
	return sb.String()
}

func (p *mdP) acceptEmphasis(node *gmAst.Emphasis) ast.InlineSlice {
	kind := ast.FormatEmph
	if node.Level == 2 {
		kind = ast.FormatStrong
	}
	return ast.InlineSlice{
		&ast.FormatNode{
			Kind:    kind,
			Attrs:   nil, //TODO
			Inlines: p.acceptChildren(node),
		},
	}
}

func (p *mdP) acceptLink(node *gmAst.Link) ast.InlineSlice {
	ref := ast.ParseReference(cleanText(string(node.Destination), true))
	var attrs *ast.Attributes
	if title := string(node.Title); len(title) > 0 {
		attrs = attrs.Set("title", cleanText(title, true))
	}
	return ast.InlineSlice{
		&ast.LinkNode{
			Ref:     ref,
			Inlines: p.acceptChildren(node),
			OnlyRef: false,
			Attrs:   attrs,
		},
	}
}

func (p *mdP) acceptImage(node *gmAst.Image) ast.InlineSlice {
	ref := ast.ParseReference(cleanText(string(node.Destination), true))
	var attrs *ast.Attributes
	if title := string(node.Title); len(title) > 0 {
		attrs = attrs.Set("title", cleanText(title, true))
	}
	return ast.InlineSlice{
		&ast.EmbedNode{
			Material: &ast.ReferenceMaterialNode{Ref: ref},
			Inlines:  &ast.InlineListNode{List: p.flattenInlineSlice(node)},
			Attrs:    attrs,
		},
	}
}

func (p *mdP) flattenInlineSlice(node gmAst.Node) ast.InlineSlice {
	ins := p.acceptChildren(node)
	var sb strings.Builder
	_, err := p.textEnc.WriteInlines(&sb, ins.List)
	if err != nil {
		panic(err)
	}
	text := sb.String()
	if text == "" {
		return nil
	}
	return ast.InlineSlice{
		&ast.TextNode{
			Text: text,
		},
	}
}

func (p *mdP) acceptAutoLink(node *gmAst.AutoLink) ast.InlineSlice {
	url := node.URL(p.source)
	if node.AutoLinkType == gmAst.AutoLinkEmail &&
		!bytes.HasPrefix(bytes.ToLower(url), []byte("mailto:")) {
		url = append([]byte("mailto:"), url...)
	}
	ref := ast.ParseReference(cleanText(string(url), false))
	label := node.Label(p.source)
	if len(label) == 0 {
		label = url
	}
	return ast.InlineSlice{
		&ast.LinkNode{
			Ref:     ref,
			Inlines: &ast.InlineListNode{List: []ast.InlineNode{&ast.TextNode{Text: string(label)}}},
			OnlyRef: true,
			Attrs:   nil, //TODO
		},
	}
}

func (p *mdP) acceptRawHTML(node *gmAst.RawHTML) ast.InlineSlice {
	segs := make([]string, 0, node.Segments.Len())
	for i := 0; i < node.Segments.Len(); i++ {
		segment := node.Segments.At(i)
		segs = append(segs, string(segment.Value(p.source)))
	}
	return ast.InlineSlice{
		&ast.LiteralNode{
			Kind:  ast.LiteralHTML,
			Attrs: nil, // TODO: add HTML as language
			Text:  strings.Join(segs, ""),
		},
	}
}
