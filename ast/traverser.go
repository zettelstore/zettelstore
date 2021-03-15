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

// A traverser is a Visitor that just traverses the AST and delegates node
// spacific actions to a Visitor. This Visitor should not traverse the AST.

// TopDownTraverser visits first the node and then the children nodes.
type TopDownTraverser struct {
	v Visitor
}

// NewTopDownTraverser creates a new traverser.
func NewTopDownTraverser(visitor Visitor) TopDownTraverser {
	return TopDownTraverser{visitor}
}

// VisitVerbatim has nothing to traverse.
func (t TopDownTraverser) VisitVerbatim(vn *VerbatimNode) { t.v.VisitVerbatim(vn) }

// VisitRegion traverses the content and the additional text.
func (t TopDownTraverser) VisitRegion(rn *RegionNode) {
	t.v.VisitRegion(rn)
	t.VisitBlockSlice(rn.Blocks)
	t.VisitInlineSlice(rn.Inlines)
}

// VisitHeading traverses the heading.
func (t TopDownTraverser) VisitHeading(hn *HeadingNode) {
	t.v.VisitHeading(hn)
	t.VisitInlineSlice(hn.Inlines)
}

// VisitHRule traverses nothing.
func (t TopDownTraverser) VisitHRule(hn *HRuleNode) { t.v.VisitHRule(hn) }

// VisitNestedList traverses all nested list elements.
func (t TopDownTraverser) VisitNestedList(ln *NestedListNode) {
	t.v.VisitNestedList(ln)
	for _, item := range ln.Items {
		t.visitItemSlice(item)
	}
}

// VisitDescriptionList traverses all description terms and their associated
// descriptions.
func (t TopDownTraverser) VisitDescriptionList(dn *DescriptionListNode) {
	t.v.VisitDescriptionList(dn)
	for _, defs := range dn.Descriptions {
		t.VisitInlineSlice(defs.Term)
		for _, descr := range defs.Descriptions {
			t.visitDescriptionSlice(descr)
		}
	}
}

// VisitPara traverses the inlines of a paragraph.
func (t TopDownTraverser) VisitPara(pn *ParaNode) {
	t.v.VisitPara(pn)
	t.VisitInlineSlice(pn.Inlines)
}

// VisitTable traverses all cells of the header and then row-wise all cells of
// the table body.
func (t TopDownTraverser) VisitTable(tn *TableNode) {
	t.v.VisitTable(tn)
	for _, col := range tn.Header {
		t.VisitInlineSlice(col.Inlines)
	}
	for _, row := range tn.Rows {
		for _, col := range row {
			t.VisitInlineSlice(col.Inlines)
		}
	}
}

// VisitBLOB traverses nothing.
func (t TopDownTraverser) VisitBLOB(bn *BLOBNode) { t.v.VisitBLOB(bn) }

// VisitText traverses nothing.
func (t TopDownTraverser) VisitText(tn *TextNode) { t.v.VisitText(tn) }

// VisitTag traverses nothing.
func (t TopDownTraverser) VisitTag(tn *TagNode) { t.v.VisitTag(tn) }

// VisitSpace traverses nothing.
func (t TopDownTraverser) VisitSpace(sn *SpaceNode) { t.v.VisitSpace(sn) }

// VisitBreak traverses nothing.
func (t TopDownTraverser) VisitBreak(bn *BreakNode) { t.v.VisitBreak(bn) }

// VisitLink traverses the link text.
func (t TopDownTraverser) VisitLink(ln *LinkNode) {
	t.v.VisitLink(ln)
	t.VisitInlineSlice(ln.Inlines)
}

// VisitImage traverses the image text.
func (t TopDownTraverser) VisitImage(in *ImageNode) {
	t.v.VisitImage(in)
	t.VisitInlineSlice(in.Inlines)
}

// VisitCite traverses the cite text.
func (t TopDownTraverser) VisitCite(cn *CiteNode) {
	t.v.VisitCite(cn)
	t.VisitInlineSlice(cn.Inlines)
}

// VisitFootnote traverses the footnote text.
func (t TopDownTraverser) VisitFootnote(fn *FootnoteNode) {
	t.v.VisitFootnote(fn)
	t.VisitInlineSlice(fn.Inlines)
}

// VisitMark traverses nothing.
func (t TopDownTraverser) VisitMark(mn *MarkNode) { t.v.VisitMark(mn) }

// VisitFormat traverses the formatted text.
func (t TopDownTraverser) VisitFormat(fn *FormatNode) {
	t.v.VisitFormat(fn)
	t.VisitInlineSlice(fn.Inlines)
}

// VisitLiteral traverses nothing.
func (t TopDownTraverser) VisitLiteral(ln *LiteralNode) { t.v.VisitLiteral(ln) }

// VisitBlockSlice traverses a block slice.
func (t TopDownTraverser) VisitBlockSlice(bns BlockSlice) {
	for _, bn := range bns {
		bn.Accept(t)
	}
}

func (t TopDownTraverser) visitItemSlice(ins ItemSlice) {
	for _, in := range ins {
		in.Accept(t)
	}
}

func (t TopDownTraverser) visitDescriptionSlice(dns DescriptionSlice) {
	for _, dn := range dns {
		dn.Accept(t)
	}
}

// VisitInlineSlice traverses a block slice.
func (t TopDownTraverser) VisitInlineSlice(ins InlineSlice) {
	for _, in := range ins {
		in.Accept(t)
	}
}
