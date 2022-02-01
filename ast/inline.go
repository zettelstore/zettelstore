//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package ast

// Definitions of inline nodes.

// InlineListNode is a list of BlockNodes.
type InlineListNode struct {
	List []InlineNode
}

func (*InlineListNode) inlineNode() { /* Just a marker */ }

// CreateInlineListNode make a new inline list node from nodes
func CreateInlineListNode(nodes ...InlineNode) *InlineListNode {
	return &InlineListNode{List: nodes}
}

// CreateInlineListNodeFromWords makes a new inline list from words,
// that will be space-separated.
func CreateInlineListNodeFromWords(words ...string) *InlineListNode {
	inl := make([]InlineNode, 0, 2*len(words)-1)
	for i, word := range words {
		if i > 0 {
			inl = append(inl, &SpaceNode{Lexeme: " "})
		}
		inl = append(inl, &TextNode{Text: word})
	}
	return &InlineListNode{List: inl}
}

// WalkChildren walks down to the list.
func (iln *InlineListNode) WalkChildren(v Visitor) {
	for _, bn := range iln.List {
		Walk(v, bn)
	}
}

// IsEmpty returns true if the list has no elements.
func (iln *InlineListNode) IsEmpty() bool { return iln == nil || len(iln.List) == 0 }

// Append inline node(s) to the list.
func (iln *InlineListNode) Append(in ...InlineNode) {
	iln.List = append(iln.List, in...)
}

// --------------------------------------------------------------------------

// TextNode just contains some text.
type TextNode struct {
	Text string // The text itself.
}

func (*TextNode) inlineNode() { /* Just a marker */ }

// WalkChildren does nothing.
func (*TextNode) WalkChildren(Visitor) { /* No children*/ }

// --------------------------------------------------------------------------

// TagNode contains a tag.
type TagNode struct {
	Tag string // The text itself.
}

func (*TagNode) inlineNode() { /* Just a marker */ }

// WalkChildren does nothing.
func (*TagNode) WalkChildren(Visitor) { /* No children*/ }

// --------------------------------------------------------------------------

// SpaceNode tracks inter-word space characters.
type SpaceNode struct {
	Lexeme string
}

func (*SpaceNode) inlineNode() { /* Just a marker */ }

// WalkChildren does nothing.
func (*SpaceNode) WalkChildren(Visitor) { /* No children*/ }

// --------------------------------------------------------------------------

// BreakNode signals a new line that must / should be interpreted as a new line break.
type BreakNode struct {
	Hard bool // Hard line break?
}

func (*BreakNode) inlineNode() { /* Just a marker */ }

// WalkChildren does nothing.
func (*BreakNode) WalkChildren(Visitor) { /* No children*/ }

// --------------------------------------------------------------------------

// LinkNode contains the specified link.
type LinkNode struct {
	Ref     *Reference
	Inlines *InlineListNode // The text associated with the link.
	OnlyRef bool            // True if no text was specified.
	Attrs   *Attributes     // Optional attributes
}

func (*LinkNode) inlineNode() { /* Just a marker */ }

// WalkChildren walks to the link text.
func (ln *LinkNode) WalkChildren(v Visitor) {
	if iln := ln.Inlines; iln != nil {
		Walk(v, iln)
	}
}

// --------------------------------------------------------------------------

// EmbedRefNode contains the specified embedded reference material.
type EmbedRefNode struct {
	Ref     *Reference      // The reference to be embedded.
	Inlines *InlineListNode // Optional text associated with the image.
	Attrs   *Attributes     // Optional attributes
}

func (*EmbedRefNode) inlineNode()      { /* Just a marker */ }
func (*EmbedRefNode) inlineEmbedNode() { /* Just a marker */ }

// WalkChildren walks to the text that describes the embedded material.
func (en *EmbedRefNode) WalkChildren(v Visitor) {
	if iln := en.Inlines; iln != nil {
		Walk(v, iln)
	}
}

// --------------------------------------------------------------------------

// EmbedBLOBNode contains the specified embedded BLOB material.
type EmbedBLOBNode struct {
	Blob    []byte          // BLOB data itself.
	Syntax  string          // Syntax of Blob
	Inlines *InlineListNode // Optional text associated with the image.
	Attrs   *Attributes     // Optional attributes
}

func (*EmbedBLOBNode) inlineNode()      { /* Just a marker */ }
func (*EmbedBLOBNode) inlineEmbedNode() { /* Just a marker */ }

// WalkChildren walks to the text that describes the embedded material.
func (en *EmbedBLOBNode) WalkChildren(v Visitor) {
	if iln := en.Inlines; iln != nil {
		Walk(v, iln)
	}
}

// --------------------------------------------------------------------------

// CiteNode contains the specified citation.
type CiteNode struct {
	Key     string          // The citation key
	Inlines *InlineListNode // Optional text associated with the citation.
	Attrs   *Attributes     // Optional attributes
}

func (*CiteNode) inlineNode() { /* Just a marker */ }

// WalkChildren walks to the cite text.
func (cn *CiteNode) WalkChildren(v Visitor) {
	if iln := cn.Inlines; iln != nil {
		Walk(v, iln)
	}
}

// --------------------------------------------------------------------------

// MarkNode contains the specified merked position.
// It is a BlockNode too, because although it is typically parsed during inline
// mode, it is moved into block mode afterwards.
type MarkNode struct {
	Text     string
	Slug     string // Slugified form of Text
	Fragment string // Unique form of Slug
}

func (*MarkNode) inlineNode() { /* Just a marker */ }

// WalkChildren does nothing.
func (*MarkNode) WalkChildren(Visitor) { /* No children*/ }

// --------------------------------------------------------------------------

// FootnoteNode contains the specified footnote.
type FootnoteNode struct {
	Inlines *InlineListNode // The footnote text.
	Attrs   *Attributes     // Optional attributes
}

func (*FootnoteNode) inlineNode() { /* Just a marker */ }

// WalkChildren walks to the footnote text.
func (fn *FootnoteNode) WalkChildren(v Visitor) {
	Walk(v, fn.Inlines)
}

// --------------------------------------------------------------------------

// FormatNode specifies some inline formatting.
type FormatNode struct {
	Kind    FormatKind
	Attrs   *Attributes // Optional attributes.
	Inlines *InlineListNode
}

// FormatKind specifies the format that is applied to the inline nodes.
type FormatKind uint8

// Constants for FormatCode
const (
	_               FormatKind = iota
	FormatEmph                 // Emphasized text.
	FormatStrong               // Strongly emphasized text.
	FormatInsert               // Inserted text.
	FormatDelete               // Deleted text.
	FormatSuper                // Superscripted text.
	FormatSub                  // SubscriptedText.
	FormatQuote                // Quoted text.
	FormatQuotation            // Quotation text.
	FormatSpan                 // Generic inline container.
	FormatMonospace            // Monospaced text.
)

func (*FormatNode) inlineNode() { /* Just a marker */ }

// WalkChildren walks to the formatted text.
func (fn *FormatNode) WalkChildren(v Visitor) {
	Walk(v, fn.Inlines)
}

// --------------------------------------------------------------------------

// LiteralNode specifies some uninterpreted text.
type LiteralNode struct {
	Kind    LiteralKind
	Attrs   *Attributes // Optional attributes.
	Content []byte
}

// LiteralKind specifies the format that is applied to code inline nodes.
type LiteralKind uint8

// Constants for LiteralCode
const (
	_              LiteralKind = iota
	LiteralZettel              // Zettel content
	LiteralProg                // Inline program code
	LiteralKeyb                // Keyboard strokes
	LiteralOutput              // Sample output.
	LiteralComment             // Inline comment
	LiteralHTML                // Inline HTML, e.g. for Markdown
)

func (*LiteralNode) inlineNode() { /* Just a marker */ }

// WalkChildren does nothing.
func (*LiteralNode) WalkChildren(Visitor) { /* No children*/ }
