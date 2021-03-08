//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package indexer allows to search for metadata and content.
package indexer

import (
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/id"
)

func collectZettelIndexData(zn *ast.ZettelNode) id.Set {
	ixv := ixVisitor{refs: id.NewSet()}
	ast.NewTopDownTraverser(&ixv).VisitBlockSlice(zn.Ast)
	return ixv.refs
}

type ixVisitor struct {
	refs id.Set
}

// VisitVerbatim does nothing.
func (lv *ixVisitor) VisitVerbatim(vn *ast.VerbatimNode) {}

// VisitRegion does nothing.
func (lv *ixVisitor) VisitRegion(rn *ast.RegionNode) {}

// VisitHeading does nothing.
func (lv *ixVisitor) VisitHeading(hn *ast.HeadingNode) {}

// VisitHRule does nothing.
func (lv *ixVisitor) VisitHRule(hn *ast.HRuleNode) {}

// VisitList does nothing.
func (lv *ixVisitor) VisitNestedList(ln *ast.NestedListNode) {}

// VisitDescriptionList does nothing.
func (lv *ixVisitor) VisitDescriptionList(dn *ast.DescriptionListNode) {}

// VisitPara does nothing.
func (lv *ixVisitor) VisitPara(pn *ast.ParaNode) {}

// VisitTable does nothing.
func (lv *ixVisitor) VisitTable(tn *ast.TableNode) {}

// VisitBLOB does nothing.
func (lv *ixVisitor) VisitBLOB(bn *ast.BLOBNode) {}

// VisitText does nothing.
func (lv *ixVisitor) VisitText(tn *ast.TextNode) {}

// VisitTag does nothing.
func (lv *ixVisitor) VisitTag(tn *ast.TagNode) {}

// VisitSpace does nothing.
func (lv *ixVisitor) VisitSpace(sn *ast.SpaceNode) {}

// VisitBreak does nothing.
func (lv *ixVisitor) VisitBreak(bn *ast.BreakNode) {}

// VisitLink collects the given link as a reference.
func (lv *ixVisitor) VisitLink(ln *ast.LinkNode) {
	ref := ln.Ref
	if ref == nil || !ref.IsZettel() {
		return
	}
	if zid, err := id.Parse(ref.URL.Path); err == nil {
		lv.refs[zid] = true
	}
}

// VisitImage collects the image links as a reference.
func (lv *ixVisitor) VisitImage(in *ast.ImageNode) {
	ref := in.Ref
	if ref == nil || !ref.IsZettel() {
		return
	}
	if zid, err := id.Parse(ref.URL.Path); err == nil {
		lv.refs[zid] = true
	}
}

// VisitCite does nothing.
func (lv *ixVisitor) VisitCite(cn *ast.CiteNode) {}

// VisitFootnote does nothing.
func (lv *ixVisitor) VisitFootnote(fn *ast.FootnoteNode) {}

// VisitMark does nothing.
func (lv *ixVisitor) VisitMark(mn *ast.MarkNode) {}

// VisitFormat does nothing.
func (lv *ixVisitor) VisitFormat(fn *ast.FormatNode) {}

// VisitLiteral does nothing.
func (lv *ixVisitor) VisitLiteral(ln *ast.LiteralNode) {}
