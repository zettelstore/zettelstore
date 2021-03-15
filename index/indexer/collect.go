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
	"zettelstore.de/z/index"
	"zettelstore.de/z/strfun"
)

func collectZettelIndexData(zn *ast.ZettelNode, refs id.Set, words index.WordSet) {
	ixv := ixVisitor{refs: refs, words: words}
	ast.NewTopDownTraverser(&ixv).VisitBlockSlice(zn.Ast)
}

func collectInlineIndexData(ins ast.InlineSlice, refs id.Set, words index.WordSet) {
	ixv := ixVisitor{refs: refs, words: words}
	ast.NewTopDownTraverser(&ixv).VisitInlineSlice(ins)
}

type ixVisitor struct {
	refs  id.Set
	words index.WordSet
}

// VisitVerbatim collects the verbatim text in the word set.
func (lv *ixVisitor) VisitVerbatim(vn *ast.VerbatimNode) {
	for _, line := range vn.Lines {
		lv.addText(line)
	}
}

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

// VisitText collects the text in the word set.
func (lv *ixVisitor) VisitText(tn *ast.TextNode) {
	lv.addText(tn.Text)
}

// VisitTag collects the tag name in the word set.
func (lv *ixVisitor) VisitTag(tn *ast.TagNode) {
	lv.addText(tn.Tag)
}

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

// VisitLiteral collects the literal words in the word set.
func (lv *ixVisitor) VisitLiteral(ln *ast.LiteralNode) {
	lv.addText(ln.Text)
}

func (lv *ixVisitor) addText(s string) {
	for _, word := range strfun.NormalizeWords(s) {
		lv.words[word] = lv.words[word] + 1
	}
}
