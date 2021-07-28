//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package ast provides the abstract syntax tree.
package ast

// Definitions of inline nodes.

// TextNode just contains some text.
type TextNode struct {
	Text string // The text itself.
}

func (tn *TextNode) inlineNode() { /* Just a marker */ }

// WalkChildren does nothing.
func (tn *TextNode) WalkChildren(v Visitor) { /* No children*/ }

// --------------------------------------------------------------------------

// TagNode contains a tag.
type TagNode struct {
	Tag string // The text itself.
}

func (tn *TagNode) inlineNode() { /* Just a marker */ }

// WalkChildren does nothing.
func (tn *TagNode) WalkChildren(v Visitor) { /* No children*/ }

// --------------------------------------------------------------------------

// SpaceNode tracks inter-word space characters.
type SpaceNode struct {
	Lexeme string
}

func (sn *SpaceNode) inlineNode() { /* Just a marker */ }

// WalkChildren does nothing.
func (sn *SpaceNode) WalkChildren(v Visitor) { /* No children*/ }

// --------------------------------------------------------------------------

// BreakNode signals a new line that must / should be interpreted as a new line break.
type BreakNode struct {
	Hard bool // Hard line break?
}

func (bn *BreakNode) inlineNode() { /* Just a marker */ }

// WalkChildren does nothing.
func (bn *BreakNode) WalkChildren(v Visitor) { /* No children*/ }

// --------------------------------------------------------------------------

// LinkNode contains the specified link.
type LinkNode struct {
	Ref     *Reference
	Inlines InlineSlice // The text associated with the link.
	OnlyRef bool        // True if no text was specified.
	Attrs   *Attributes // Optional attributes
}

func (ln *LinkNode) inlineNode() { /* Just a marker */ }

// WalkChildren walks to the link text.
func (ln *LinkNode) WalkChildren(v Visitor) {
	WalkInlineSlice(v, ln.Inlines)
}

// --------------------------------------------------------------------------

// EmbedNode contains the specified image reference.
type EmbedNode struct {
	Ref     *Reference  // Reference to image
	Blob    []byte      // BLOB data of the image, as an alternative to Ref.
	Syntax  string      // Syntax of Blob
	Inlines InlineSlice // The text associated with the image.
	Attrs   *Attributes // Optional attributes
}

func (in *EmbedNode) inlineNode() { /* Just a marker */ }

// WalkChildren walks to the text that describes the embedded material.
func (in *EmbedNode) WalkChildren(v Visitor) {
	WalkInlineSlice(v, in.Inlines)
}

// --------------------------------------------------------------------------

// CiteNode contains the specified citation.
type CiteNode struct {
	Key     string      // The citation key
	Inlines InlineSlice // The text associated with the citation.
	Attrs   *Attributes // Optional attributes
}

func (cn *CiteNode) inlineNode() { /* Just a marker */ }

// WalkChildren walks to the cite text.
func (cn *CiteNode) WalkChildren(v Visitor) {
	WalkInlineSlice(v, cn.Inlines)
}

// --------------------------------------------------------------------------

// MarkNode contains the specified merked position.
// It is a BlockNode too, because although it is typically parsed during inline
// mode, it is moved into block mode afterwards.
type MarkNode struct {
	Text string
}

func (mn *MarkNode) inlineNode() { /* Just a marker */ }

// WalkChildren does nothing.
func (mn *MarkNode) WalkChildren(v Visitor) { /* No children*/ }

// --------------------------------------------------------------------------

// FootnoteNode contains the specified footnote.
type FootnoteNode struct {
	Inlines InlineSlice // The footnote text.
	Attrs   *Attributes // Optional attributes
}

func (fn *FootnoteNode) inlineNode() { /* Just a marker */ }

// WalkChildren walks to the footnote text.
func (fn *FootnoteNode) WalkChildren(v Visitor) {
	WalkInlineSlice(v, fn.Inlines)
}

// --------------------------------------------------------------------------

// FormatNode specifies some inline formatting.
type FormatNode struct {
	Kind    FormatKind
	Attrs   *Attributes // Optional attributes.
	Inlines InlineSlice
}

// FormatKind specifies the format that is applied to the inline nodes.
type FormatKind uint8

// Constants for FormatCode
const (
	_               FormatKind = iota
	FormatItalic               // Italic text.
	FormatEmph                 // Semantically emphasized text.
	FormatBold                 // Bold text.
	FormatStrong               // Semantically strongly emphasized text.
	FormatUnder                // Underlined text.
	FormatInsert               // Inserted text.
	FormatStrike               // Text that is no longer relevant or no longer accurate.
	FormatDelete               // Deleted text.
	FormatSuper                // Superscripted text.
	FormatSub                  // SubscriptedText.
	FormatQuote                // Quoted text.
	FormatQuotation            // Quotation text.
	FormatSmall                // Smaller text.
	FormatSpan                 // Generic inline container.
	FormatMonospace            // Monospaced text.
)

func (fn *FormatNode) inlineNode() { /* Just a marker */ }

// WalkChildren walks to the formatted text.
func (fn *FormatNode) WalkChildren(v Visitor) {
	WalkInlineSlice(v, fn.Inlines)
}

// --------------------------------------------------------------------------

// LiteralNode specifies some uninterpreted text.
type LiteralNode struct {
	Kind  LiteralKind
	Attrs *Attributes // Optional attributes.
	Text  string
}

// LiteralKind specifies the format that is applied to code inline nodes.
type LiteralKind uint8

// Constants for LiteralCode
const (
	_              LiteralKind = iota
	LiteralProg                // Inline program code.
	LiteralKeyb                // Keyboard strokes.
	LiteralOutput              // Sample output.
	LiteralComment             // Inline comment
	LiteralHTML                // Inline HTML, e.g. for Markdown
)

func (ln *LiteralNode) inlineNode() { /* Just a marker */ }

// WalkChildren does nothing.
func (ln *LiteralNode) WalkChildren(v Visitor) { /* No children*/ }
