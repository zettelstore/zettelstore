//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
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

func (tn *TextNode) inlineNode() {}

// Accept a visitor and visit the node.
func (tn *TextNode) Accept(v Visitor) { v.VisitText(tn) }

// --------------------------------------------------------------------------

// TagNode contains a tag.
type TagNode struct {
	Tag string // The text itself.
}

func (tn *TagNode) inlineNode() {}

// Accept a visitor and visit the node.
func (tn *TagNode) Accept(v Visitor) { v.VisitTag(tn) }

// --------------------------------------------------------------------------

// SpaceNode tracks inter-word space characters.
type SpaceNode struct {
	Lexeme string
}

func (sn *SpaceNode) inlineNode() {}

// Accept a visitor and visit the node.
func (sn *SpaceNode) Accept(v Visitor) { v.VisitSpace(sn) }

// --------------------------------------------------------------------------

// BreakNode signals a new line that must / should be interpreted as a new line break.
type BreakNode struct {
	Hard bool // Hard line break?
}

func (bn *BreakNode) inlineNode() {}

// Accept a visitor and visit the node.
func (bn *BreakNode) Accept(v Visitor) { v.VisitBreak(bn) }

// --------------------------------------------------------------------------

// LinkNode contains the specified link.
type LinkNode struct {
	Ref     *Reference
	Inlines InlineSlice // The text associated with the link.
	OnlyRef bool        // True if no text was specified.
	Attrs   *Attributes // Optional attributes
}

func (ln *LinkNode) inlineNode() {}

// Accept a visitor and visit the node.
func (ln *LinkNode) Accept(v Visitor) { v.VisitLink(ln) }

// --------------------------------------------------------------------------

// ImageNode contains the specified image reference.
type ImageNode struct {
	Ref     *Reference  // Reference to image
	Blob    []byte      // BLOB data of the image, as an alternative to Ref.
	Syntax  string      // Syntax of Blob
	Inlines InlineSlice // The text associated with the image.
	Attrs   *Attributes // Optional attributes
}

func (in *ImageNode) inlineNode() {}

// Accept a visitor and visit the node.
func (in *ImageNode) Accept(v Visitor) { v.VisitImage(in) }

// --------------------------------------------------------------------------

// CiteNode contains the specified citation.
type CiteNode struct {
	Key     string      // The citation key
	Inlines InlineSlice // The text associated with the citation.
	Attrs   *Attributes // Optional attributes
}

func (cn *CiteNode) inlineNode() {}

// Accept a visitor and visit the node.
func (cn *CiteNode) Accept(v Visitor) { v.VisitCite(cn) }

// --------------------------------------------------------------------------

// MarkNode contains the specified merked position.
// It is a BlockNode too, because although it is typically parsed during inline
// mode, it is moved into block mode afterwards.
type MarkNode struct {
	Text string
}

func (mn *MarkNode) inlineNode() {}

// Accept a visitor and visit the node.
func (mn *MarkNode) Accept(v Visitor) { v.VisitMark(mn) }

// --------------------------------------------------------------------------

// FootnoteNode contains the specified footnote.
type FootnoteNode struct {
	Inlines InlineSlice // The footnote text.
	Attrs   *Attributes // Optional attributes
}

func (fn *FootnoteNode) inlineNode() {}

// Accept a visitor and visit the node.
func (fn *FootnoteNode) Accept(v Visitor) { v.VisitFootnote(fn) }

// --------------------------------------------------------------------------

// FormatNode specifies some inline formatting.
type FormatNode struct {
	Code    FormatCode
	Attrs   *Attributes // Optional attributes.
	Inlines InlineSlice
}

// FormatCode specifies the format that is applied to the inline nodes.
type FormatCode int

// Constants for FormatCode
const (
	_               FormatCode = iota
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

func (fn *FormatNode) inlineNode() {}

// Accept a visitor and visit the node.
func (fn *FormatNode) Accept(v Visitor) { v.VisitFormat(fn) }

// --------------------------------------------------------------------------

// LiteralNode specifies some uninterpreted text.
type LiteralNode struct {
	Code  LiteralCode
	Attrs *Attributes // Optional attributes.
	Text  string
}

// LiteralCode specifies the format that is applied to code inline nodes.
type LiteralCode int

// Constants for LiteralCode
const (
	_              LiteralCode = iota
	LiteralProg                // Inline program code.
	LiteralKeyb                // Keyboard strokes.
	LiteralOutput              // Sample output.
	LiteralComment             // Inline comment
	LiteralHTML                // Inline HTML, e.g. for Markdown
)

func (rn *LiteralNode) inlineNode() {}

// Accept a visitor and visit the node.
func (rn *LiteralNode) Accept(v Visitor) { v.VisitLiteral(rn) }
