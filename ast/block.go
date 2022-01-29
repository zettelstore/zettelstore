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

// Definition of Block nodes.

// BlockListNode is a list of BlockNodes.
type BlockListNode struct {
	List []BlockNode
}

func (*BlockListNode) blockNode() { /* Just a marker */ }

// WalkChildren walks down to the descriptions.
func (bln *BlockListNode) WalkChildren(v Visitor) {
	for _, bn := range bln.List {
		Walk(v, bn)
	}
}

//--------------------------------------------------------------------------

// ParaNode contains just a sequence of inline elements.
// Another name is "paragraph".
type ParaNode struct {
	Inlines *InlineListNode
}

func (*ParaNode) blockNode()       { /* Just a marker */ }
func (*ParaNode) itemNode()        { /* Just a marker */ }
func (*ParaNode) descriptionNode() { /* Just a marker */ }

// NewParaNode creates an empty ParaNode.
func NewParaNode() *ParaNode { return &ParaNode{Inlines: &InlineListNode{}} }

// WalkChildren walks down the inline elements.
func (pn *ParaNode) WalkChildren(v Visitor) {
	Walk(v, pn.Inlines)
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

func (*VerbatimNode) blockNode() { /* Just a marker */ }
func (*VerbatimNode) itemNode()  { /* Just a marker */ }

// WalkChildren does nothing.
func (*VerbatimNode) WalkChildren(Visitor) { /* No children*/ }

//--------------------------------------------------------------------------

// RegionNode encapsulates a region of block nodes.
type RegionNode struct {
	Kind    RegionKind
	Attrs   *Attributes
	Blocks  *BlockListNode
	Inlines *InlineListNode // Optional text at the end of the region
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

func (*RegionNode) blockNode() { /* Just a marker */ }
func (*RegionNode) itemNode()  { /* Just a marker */ }

// WalkChildren walks down the blocks and the text.
func (rn *RegionNode) WalkChildren(v Visitor) {
	Walk(v, rn.Blocks)
	if iln := rn.Inlines; iln != nil {
		Walk(v, iln)
	}
}

//--------------------------------------------------------------------------

// HeadingNode stores the heading text and level.
type HeadingNode struct {
	Level    int
	Inlines  *InlineListNode // Heading text, possibly formatted
	Slug     string          // Heading text, normalized
	Fragment string          // Heading text, suitable to be used as an unique URL fragment
	Attrs    *Attributes
}

func (*HeadingNode) blockNode() { /* Just a marker */ }
func (*HeadingNode) itemNode()  { /* Just a marker */ }

// WalkChildren walks the heading text.
func (hn *HeadingNode) WalkChildren(v Visitor) {
	Walk(v, hn.Inlines)
}

//--------------------------------------------------------------------------

// HRuleNode specifies a horizontal rule.
type HRuleNode struct {
	Attrs *Attributes
}

func (*HRuleNode) blockNode() { /* Just a marker */ }
func (*HRuleNode) itemNode()  { /* Just a marker */ }

// WalkChildren does nothing.
func (*HRuleNode) WalkChildren(Visitor) { /* No children*/ }

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

func (*NestedListNode) blockNode() { /* Just a marker */ }
func (*NestedListNode) itemNode()  { /* Just a marker */ }

// WalkChildren walks down the items.
func (ln *NestedListNode) WalkChildren(v Visitor) {
	for _, item := range ln.Items {
		WalkItemSlice(v, item)
	}
}

//--------------------------------------------------------------------------

// DescriptionListNode specifies a description list.
type DescriptionListNode struct {
	Descriptions []Description
}

// Description is one element of a description list.
type Description struct {
	Term         *InlineListNode
	Descriptions []DescriptionSlice
}

func (*DescriptionListNode) blockNode() { /* Just a marker */ }

// WalkChildren walks down to the descriptions.
func (dn *DescriptionListNode) WalkChildren(v Visitor) {
	for _, desc := range dn.Descriptions {
		Walk(v, desc.Term)
		for _, dns := range desc.Descriptions {
			WalkDescriptionSlice(v, dns)
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
	Align   Alignment       // Cell alignment
	Inlines *InlineListNode // Cell content
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

func (*TableNode) blockNode() { /* Just a marker */ }

// WalkChildren walks down to the cells.
func (tn *TableNode) WalkChildren(v Visitor) {
	for _, cell := range tn.Header {
		Walk(v, cell.Inlines)
	}
	for _, row := range tn.Rows {
		for _, cell := range row {
			Walk(v, cell.Inlines)
		}
	}
}

//--------------------------------------------------------------------------

// TranscludeNode specifies block content from other zettel to embedded in
// current zettel
type TranscludeNode struct {
	Ref *Reference
}

func (*TranscludeNode) blockNode() { /* Just a marker */ }

// WalkChildren does nothing.
func (*TranscludeNode) WalkChildren(Visitor) { /* No children*/ }

//--------------------------------------------------------------------------

// BLOBNode contains just binary data that must be interpreted according to
// a syntax.
type BLOBNode struct {
	Title  string
	Syntax string
	Blob   []byte
}

func (*BLOBNode) blockNode() { /* Just a marker */ }

// WalkChildren does nothing.
func (*BLOBNode) WalkChildren(Visitor) { /* No children*/ }
