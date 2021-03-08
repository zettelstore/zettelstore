//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package parser provides a generic interface to a range of different parsers.
package parser

import (
	"strconv"
	"strings"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/strfun"

	// Ensure that the text encoder is available
	_ "zettelstore.de/z/encoder/textenc"
)

func cleanupBlockSlice(bs ast.BlockSlice) {
	cv := &cleanupVisitor{
		textEnc: encoder.Create("text", nil),
		doMark:  false,
	}
	t := ast.NewTopDownTraverser(cv)
	t.VisitBlockSlice(bs)
	if cv.hasMark {
		cv.doMark = true
		t.VisitBlockSlice(bs)
	}
}

type cleanupVisitor struct {
	textEnc encoder.Encoder
	ids     map[string]ast.Node
	hasMark bool
	doMark  bool
}

// VisitVerbatim does nothing.
func (cv *cleanupVisitor) VisitVerbatim(vn *ast.VerbatimNode) {}

// VisitRegion does nothing.
func (cv *cleanupVisitor) VisitRegion(rn *ast.RegionNode) {}

// VisitHeading calculates the heading slug.
func (cv *cleanupVisitor) VisitHeading(hn *ast.HeadingNode) {
	if cv.doMark || hn == nil || hn.Inlines == nil {
		return
	}
	var sb strings.Builder
	_, err := cv.textEnc.WriteInlines(&sb, hn.Inlines)
	if err != nil {
		return
	}
	s := strfun.Slugify(sb.String())
	if len(s) > 0 {
		hn.Slug = cv.addIdentifier(s, hn)
	}
}

// VisitHRule does nothing.
func (cv *cleanupVisitor) VisitHRule(hn *ast.HRuleNode) {}

// VisitList does nothing.
func (cv *cleanupVisitor) VisitNestedList(ln *ast.NestedListNode) {}

// VisitDescriptionList does nothing.
func (cv *cleanupVisitor) VisitDescriptionList(dn *ast.DescriptionListNode) {}

// VisitPara does nothing.
func (cv *cleanupVisitor) VisitPara(pn *ast.ParaNode) {}

// VisitTable does nothing.
func (cv *cleanupVisitor) VisitTable(tn *ast.TableNode) {}

// VisitBLOB does nothing.
func (cv *cleanupVisitor) VisitBLOB(bn *ast.BLOBNode) {}

// VisitText does nothing.
func (cv *cleanupVisitor) VisitText(tn *ast.TextNode) {}

// VisitTag does nothing.
func (cv *cleanupVisitor) VisitTag(tn *ast.TagNode) {}

// VisitSpace does nothing.
func (cv *cleanupVisitor) VisitSpace(sn *ast.SpaceNode) {}

// VisitBreak does nothing.
func (cv *cleanupVisitor) VisitBreak(bn *ast.BreakNode) {}

// VisitLink collects the given link as a reference.
func (cv *cleanupVisitor) VisitLink(ln *ast.LinkNode) {}

// VisitImage collects the image links as a reference.
func (cv *cleanupVisitor) VisitImage(in *ast.ImageNode) {}

// VisitCite does nothing.
func (cv *cleanupVisitor) VisitCite(cn *ast.CiteNode) {}

// VisitFootnote does nothing.
func (cv *cleanupVisitor) VisitFootnote(fn *ast.FootnoteNode) {}

// VisitMark checks for duplicate marks and changes them.
func (cv *cleanupVisitor) VisitMark(mn *ast.MarkNode) {
	if mn == nil {
		return
	}
	if !cv.doMark {
		cv.hasMark = true
		return
	}
	if mn.Text == "" {
		mn.Text = cv.addIdentifier("*", mn)
		return
	}
	mn.Text = cv.addIdentifier(mn.Text, mn)
}

// VisitFormat does nothing.
func (cv *cleanupVisitor) VisitFormat(fn *ast.FormatNode) {}

// VisitLiteral does nothing.
func (cv *cleanupVisitor) VisitLiteral(ln *ast.LiteralNode) {}

func (cv *cleanupVisitor) addIdentifier(id string, node ast.Node) string {
	if cv.ids == nil {
		cv.ids = map[string]ast.Node{id: node}
		return id
	}
	if n, ok := cv.ids[id]; ok && n != node {
		prefix := id + "-"
		for count := 1; ; count++ {
			newID := prefix + strconv.Itoa(count)
			if n, ok := cv.ids[newID]; !ok || n == node {
				cv.ids[newID] = node
				return newID
			}
		}
	}
	cv.ids[id] = node
	return id
}
