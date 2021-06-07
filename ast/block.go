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

// Definition of Block nodes.

// ParaNode contains just a sequence of inline elements.
// Another name is "paragraph".
type ParaNode struct {
	Inlines InlineSlice
}

func (pn *ParaNode) blockNode()       {}
func (pn *ParaNode) itemNode()        {}
func (pn *ParaNode) descriptionNode() {}

// Accept a visitor and visit the node.
func (pn *ParaNode) Accept(v Visitor) { v.VisitPara(pn) }

// WalkChildren walks down the inline elements.
func (pn *ParaNode) WalkChildren(v WalkVisitor) {
	WalkInlineSlice(v, pn.Inlines)
}

//--------------------------------------------------------------------------

// VerbatimNode contains lines of uninterpreted text
type VerbatimNode struct {
	Kind  VerbatimKind
	Attrs *Attributes
	Lines []string
}

// VerbatimKind specifies the format that is applied to code inline nodes.
type VerbatimKind uint8

// Constants for VerbatimCode
const (
	_               VerbatimKind = iota
	VerbatimProg                 // Program code.
	VerbatimComment              // Block comment
	VerbatimHTML                 // Block HTML, e.g. for Markdown
)

func (vn *VerbatimNode) blockNode() {}
func (vn *VerbatimNode) itemNode()  {}

// Accept a visitor an visit the node.
func (vn *VerbatimNode) Accept(v Visitor) { v.VisitVerbatim(vn) }

// WalkChildren does nothing.
func (vn *VerbatimNode) WalkChildren(v WalkVisitor) {}

//--------------------------------------------------------------------------

// RegionNode encapsulates a region of block nodes.
type RegionNode struct {
	Kind    RegionKind
	Attrs   *Attributes
	Blocks  BlockSlice
	Inlines InlineSlice // Additional text at the end of the region
}

// RegionKind specifies the actual region type.
type RegionKind uint8

// Values for RegionCode
const (
	_           RegionKind = iota
	RegionSpan             // Just a span of blocks
	RegionQuote            // A longer quotation
	RegionVerse            // Line breaks matter
)

func (rn *RegionNode) blockNode() {}
func (rn *RegionNode) itemNode()  {}

// Accept a visitor and visit the node.
func (rn *RegionNode) Accept(v Visitor) { v.VisitRegion(rn) }

// WalkChildren walks down the blocks and the text.
func (rn *RegionNode) WalkChildren(v WalkVisitor) {
	WalkBlockSlice(v, rn.Blocks)
	WalkInlineSlice(v, rn.Inlines)
}

//--------------------------------------------------------------------------

// HeadingNode stores the heading text and level.
type HeadingNode struct {
	Level   int
	Inlines InlineSlice // Heading text, possibly formatted
	Slug    string      // Heading text, suitable to be used as an URL fragment
	Attrs   *Attributes
}

func (hn *HeadingNode) blockNode() {}
func (hn *HeadingNode) itemNode()  {}

// Accept a visitor and visit the node.
func (hn *HeadingNode) Accept(v Visitor) { v.VisitHeading(hn) }

// WalkChildren walks the heading text.
func (hn *HeadingNode) WalkChildren(v WalkVisitor) {
	WalkInlineSlice(v, hn.Inlines)
}

//--------------------------------------------------------------------------

// HRuleNode specifies a horizontal rule.
type HRuleNode struct {
	Attrs *Attributes
}

func (hn *HRuleNode) blockNode() {}
func (hn *HRuleNode) itemNode()  {}

// Accept a visitor and visit the node.
func (hn *HRuleNode) Accept(v Visitor) { v.VisitHRule(hn) }

// WalkChildren does nothing.
func (hn *HRuleNode) WalkChildren(v WalkVisitor) {}

//--------------------------------------------------------------------------

// NestedListNode specifies a nestable list, either ordered or unordered.
type NestedListNode struct {
	Kind  NestedListKind
	Items []ItemSlice
	Attrs *Attributes
}

// NestedListKind specifies the actual list type.
type NestedListKind uint8

// Values for ListCode
const (
	_                   NestedListKind = iota
	NestedListOrdered                  // Ordered list.
	NestedListUnordered                // Unordered list.
	NestedListQuote                    // Quote list.
)

func (ln *NestedListNode) blockNode() {}
func (ln *NestedListNode) itemNode()  {}

// Accept a visitor and visit the node.
func (ln *NestedListNode) Accept(v Visitor) { v.VisitNestedList(ln) }

// WalkChildren walks down the items.
func (ln *NestedListNode) WalkChildren(v WalkVisitor) {
	for _, item := range ln.Items {
		for _, in := range item {
			Walk(v, in)
		}
	}
}

//--------------------------------------------------------------------------

// DescriptionListNode specifies a description list.
type DescriptionListNode struct {
	Descriptions []Description
}

// Description is one element of a description list.
type Description struct {
	Term         InlineSlice
	Descriptions []DescriptionSlice
}

func (dn *DescriptionListNode) blockNode() {}

// Accept a visitor and visit the node.
func (dn *DescriptionListNode) Accept(v Visitor) { v.VisitDescriptionList(dn) }

// WalkChildren walks down to the descriptions.
func (dn *DescriptionListNode) WalkChildren(v WalkVisitor) {
	for _, desc := range dn.Descriptions {
		WalkInlineSlice(v, desc.Term)
		for _, dns := range desc.Descriptions {
			for _, dn := range dns {
				Walk(v, dn)
			}
		}
	}
}

//--------------------------------------------------------------------------

// TableNode specifies a full table
type TableNode struct {
	Header TableRow    // The header row
	Align  []Alignment // Default column alignment
	Rows   []TableRow  // The slice of cell rows
}

// TableCell contains the data for one table cell
type TableCell struct {
	Align   Alignment   // Cell alignment
	Inlines InlineSlice // Cell content
}

// TableRow is a slice of cells.
type TableRow []*TableCell

// Alignment specifies text alignment.
// Currently only for tables.
type Alignment int

// Constants for Alignment.
const (
	_            Alignment = iota
	AlignDefault           // Default alignment, inherited
	AlignLeft              // Left alignment
	AlignCenter            // Center the content
	AlignRight             // Right alignment
)

func (tn *TableNode) blockNode() {}

// Accept a visitor and visit the node.
func (tn *TableNode) Accept(v Visitor) { v.VisitTable(tn) }

// WalkChildren walks down to the cells.
func (tn *TableNode) WalkChildren(v WalkVisitor) {
	for _, cell := range tn.Header {
		WalkInlineSlice(v, cell.Inlines)
	}
	for _, row := range tn.Rows {
		for _, cell := range row {
			WalkInlineSlice(v, cell.Inlines)
		}
	}
}

//--------------------------------------------------------------------------

// BLOBNode contains just binary data that must be interpreted according to
// a syntax.
type BLOBNode struct {
	Title  string
	Syntax string
	Blob   []byte
}

func (bn *BLOBNode) blockNode() {}

// Accept a visitor and visit the node.
func (bn *BLOBNode) Accept(v Visitor) { v.VisitBLOB(bn) }

// WalkChildren does nothing.
func (bn *BLOBNode) WalkChildren(v WalkVisitor) {}
