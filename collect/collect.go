//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package collect provides functions to collect items from a syntax tree.
package collect

import (
	"zettelstore.de/z/ast"
)

// Summary stores the relevant parts of the syntax tree
type Summary struct {
	Links  []*ast.Reference // list of all referenced links
	Images []*ast.Reference // list of all referenced images
	Cites  []*ast.CiteNode  // list of all referenced citations
}

// References returns all references mentioned in the given zettel. This also
// includes references to images.
func References(zn *ast.ZettelNode) Summary {
	lv := linkVisitor{}
	ast.NewTopDownTraverser(&lv).VisitBlockSlice(zn.Ast)
	return lv.summary
}

type linkVisitor struct {
	summary Summary
}

// VisitVerbatim does nothing.
func (lv *linkVisitor) VisitVerbatim(vn *ast.VerbatimNode) {}

// VisitRegion does nothing.
func (lv *linkVisitor) VisitRegion(rn *ast.RegionNode) {}

// VisitHeading does nothing.
func (lv *linkVisitor) VisitHeading(hn *ast.HeadingNode) {}

// VisitHRule does nothing.
func (lv *linkVisitor) VisitHRule(hn *ast.HRuleNode) {}

// VisitList does nothing.
func (lv *linkVisitor) VisitNestedList(ln *ast.NestedListNode) {}

// VisitDescriptionList does nothing.
func (lv *linkVisitor) VisitDescriptionList(dn *ast.DescriptionListNode) {}

// VisitPara does nothing.
func (lv *linkVisitor) VisitPara(pn *ast.ParaNode) {}

// VisitTable does nothing.
func (lv *linkVisitor) VisitTable(tn *ast.TableNode) {}

// VisitBLOB does nothing.
func (lv *linkVisitor) VisitBLOB(bn *ast.BLOBNode) {}

// VisitText does nothing.
func (lv *linkVisitor) VisitText(tn *ast.TextNode) {}

// VisitTag does nothing.
func (lv *linkVisitor) VisitTag(tn *ast.TagNode) {}

// VisitSpace does nothing.
func (lv *linkVisitor) VisitSpace(sn *ast.SpaceNode) {}

// VisitBreak does nothing.
func (lv *linkVisitor) VisitBreak(bn *ast.BreakNode) {}

// VisitLink collects the given link as a reference.
func (lv *linkVisitor) VisitLink(ln *ast.LinkNode) {
	lv.summary.Links = append(lv.summary.Links, ln.Ref)
}

// VisitImage collects the image links as a reference.
func (lv *linkVisitor) VisitImage(in *ast.ImageNode) {
	if in.Ref != nil {
		lv.summary.Images = append(lv.summary.Images, in.Ref)
	}
}

// VisitCite collects the citation.
func (lv *linkVisitor) VisitCite(cn *ast.CiteNode) {
	lv.summary.Cites = append(lv.summary.Cites, cn)
}

// VisitFootnote does nothing.
func (lv *linkVisitor) VisitFootnote(fn *ast.FootnoteNode) {}

// VisitMark does nothing.
func (lv *linkVisitor) VisitMark(mn *ast.MarkNode) {}

// VisitFormat does nothing.
func (lv *linkVisitor) VisitFormat(fn *ast.FormatNode) {}

// VisitLiteral does nothing.
func (lv *linkVisitor) VisitLiteral(ln *ast.LiteralNode) {}
