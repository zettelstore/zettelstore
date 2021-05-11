//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package manager coordinates the various places and indexes of a Zettelstore.
package manager

import (
	"strings"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/place/manager/store"
	"zettelstore.de/z/strfun"
)

type collectData struct {
	refs  id.Set
	words store.WordSet
	urls  store.WordSet
}

func (data *collectData) initialize() {
	data.refs = id.NewSet()
	data.words = store.NewWordSet()
	data.urls = store.NewWordSet()
}

func collectZettelIndexData(zn *ast.ZettelNode, data *collectData) {
	ast.NewTopDownTraverser(data).VisitBlockSlice(zn.Ast)
}

func collectInlineIndexData(ins ast.InlineSlice, data *collectData) {
	ast.NewTopDownTraverser(data).VisitInlineSlice(ins)
}

// VisitVerbatim collects the verbatim text in the word set.
func (data *collectData) VisitVerbatim(vn *ast.VerbatimNode) {
	for _, line := range vn.Lines {
		data.addText(line)
	}
}

// VisitRegion does nothing.
func (data *collectData) VisitRegion(rn *ast.RegionNode) {}

// VisitHeading does nothing.
func (data *collectData) VisitHeading(hn *ast.HeadingNode) {}

// VisitHRule does nothing.
func (data *collectData) VisitHRule(hn *ast.HRuleNode) {}

// VisitList does nothing.
func (data *collectData) VisitNestedList(ln *ast.NestedListNode) {}

// VisitDescriptionList does nothing.
func (data *collectData) VisitDescriptionList(dn *ast.DescriptionListNode) {}

// VisitPara does nothing.
func (data *collectData) VisitPara(pn *ast.ParaNode) {}

// VisitTable does nothing.
func (data *collectData) VisitTable(tn *ast.TableNode) {}

// VisitBLOB does nothing.
func (data *collectData) VisitBLOB(bn *ast.BLOBNode) {}

// VisitText collects the text in the word set.
func (data *collectData) VisitText(tn *ast.TextNode) {
	data.addText(tn.Text)
}

// VisitTag collects the tag name in the word set.
func (data *collectData) VisitTag(tn *ast.TagNode) {
	data.addText(tn.Tag)
}

// VisitSpace does nothing.
func (data *collectData) VisitSpace(sn *ast.SpaceNode) {}

// VisitBreak does nothing.
func (data *collectData) VisitBreak(bn *ast.BreakNode) {}

// VisitLink collects the given link as a reference.
func (data *collectData) VisitLink(ln *ast.LinkNode) {
	ref := ln.Ref
	if ref == nil {
		return
	}
	if ref.IsExternal() {
		data.urls.Add(strings.ToLower(ref.Value))
	}
	if !ref.IsZettel() {
		return
	}
	if zid, err := id.Parse(ref.URL.Path); err == nil {
		data.refs[zid] = true
	}
}

// VisitImage collects the image links as a reference.
func (data *collectData) VisitImage(in *ast.ImageNode) {
	ref := in.Ref
	if ref == nil {
		return
	}
	if ref.IsExternal() {
		data.urls.Add(strings.ToLower(ref.Value))
	}
	if !ref.IsZettel() {
		return
	}
	if zid, err := id.Parse(ref.URL.Path); err == nil {
		data.refs[zid] = true
	}
}

// VisitCite does nothing.
func (data *collectData) VisitCite(cn *ast.CiteNode) {}

// VisitFootnote does nothing.
func (data *collectData) VisitFootnote(fn *ast.FootnoteNode) {}

// VisitMark does nothing.
func (data *collectData) VisitMark(mn *ast.MarkNode) {}

// VisitFormat does nothing.
func (data *collectData) VisitFormat(fn *ast.FormatNode) {}

// VisitLiteral collects the literal words in the word set.
func (data *collectData) VisitLiteral(ln *ast.LiteralNode) {
	data.addText(ln.Text)
}

func (data *collectData) addText(s string) {
	for _, word := range strfun.NormalizeWords(s) {
		data.words.Add(word)
	}
}
