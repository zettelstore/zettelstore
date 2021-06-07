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

// Visitor is the interface all visitors must implement.
type Visitor interface {
	// Block nodes
	VisitPara(pn *ParaNode)
	VisitVerbatim(vn *VerbatimNode)
	VisitRegion(rn *RegionNode)
	VisitHeading(hn *HeadingNode)
	VisitHRule(hn *HRuleNode)
	VisitNestedList(ln *NestedListNode)
	VisitDescriptionList(dn *DescriptionListNode)
	VisitTable(tn *TableNode)
	VisitBLOB(bn *BLOBNode)

	// Inline nodes
	VisitText(tn *TextNode)
	VisitTag(tn *TagNode)
	VisitSpace(sn *SpaceNode)
	VisitBreak(bn *BreakNode)
	VisitLink(ln *LinkNode)
	VisitImage(in *ImageNode)
	VisitCite(cn *CiteNode)
	VisitFootnote(fn *FootnoteNode)
	VisitMark(mn *MarkNode)
	VisitFormat(fn *FormatNode)
	VisitLiteral(ln *LiteralNode)
}
