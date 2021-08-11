//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package zettelmark provides a parser for zettelmarkup.
package zettelmark

import (
	"strings"

	"zettelstore.de/z/ast"
)

// postProcessBlocks is the entry point for post-processing a list of block nodes.
func postProcessBlocks(bs *ast.BlockListNode) {
	pp := postProcessor{}
	ast.Walk(&pp, bs)
}

// postProcessInlines is the entry point for post-processing a list of inline nodes.
func postProcessInlines(iln *ast.InlineListNode) {
	pp := postProcessor{}
	ast.Walk(&pp, iln)
}

// postProcessor is a visitor that cleans the abstract syntax tree.
type postProcessor struct {
	inVerse bool
}

func (pp *postProcessor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.BlockListNode:
		pp.visitBlockList(n)
	case *ast.InlineListNode:
		pp.visitInlineList(n)
	case *ast.ParaNode:
		return pp
	case *ast.RegionNode:
		pp.visitRegion(n)
		return pp
	case *ast.HeadingNode:
		return pp
	case *ast.NestedListNode:
		pp.visitNestedList(n)
	case *ast.DescriptionListNode:
		pp.visitDescriptionList(n)
		return pp
	case *ast.TableNode:
		pp.visitTable(n)
	case *ast.LinkNode:
		return pp
	case *ast.EmbedNode:
		return pp
	case *ast.CiteNode:
		return pp
	case *ast.FootnoteNode:
		return pp
	case *ast.FormatNode:
		pp.visitFormat(n)
		return pp
	}
	return nil
}

func (pp *postProcessor) visitRegion(rn *ast.RegionNode) {
	oldVerse := pp.inVerse
	if rn.Kind == ast.RegionVerse {
		pp.inVerse = true
	}
	pp.visitBlockList(rn.Blocks)
	pp.inVerse = oldVerse
}

func (pp *postProcessor) visitNestedList(ln *ast.NestedListNode) {
	for i, item := range ln.Items {
		ln.Items[i] = pp.processItemSlice(item)
	}
}

func (pp *postProcessor) visitDescriptionList(dn *ast.DescriptionListNode) {
	for i, def := range dn.Descriptions {
		for j, b := range def.Descriptions {
			dn.Descriptions[i].Descriptions[j] = pp.processDescriptionSlice(b)
		}
	}
}

func (pp *postProcessor) visitTable(tn *ast.TableNode) {
	width := tableWidth(tn)
	tn.Align = make([]ast.Alignment, width)
	for i := 0; i < width; i++ {
		tn.Align[i] = ast.AlignDefault
	}
	if len(tn.Rows) > 0 && isHeaderRow(tn.Rows[0]) {
		tn.Header = tn.Rows[0]
		tn.Rows = tn.Rows[1:]
		pp.visitTableHeader(tn)
	}
	if len(tn.Header) > 0 {
		tn.Header = appendCells(tn.Header, width, tn.Align)
		for i, cell := range tn.Header {
			pp.processCell(cell, tn.Align[i])
		}
	}
	pp.visitTableRows(tn, width)
}

func (pp *postProcessor) visitTableHeader(tn *ast.TableNode) {
	for pos, cell := range tn.Header {
		ins := cell.Inlines.List
		if len(ins) == 0 {
			continue
		}
		if textNode, ok := ins[0].(*ast.TextNode); ok {
			textNode.Text = strings.TrimPrefix(textNode.Text, "=")
		}
		if textNode, ok := ins[len(ins)-1].(*ast.TextNode); ok {
			if tnl := len(textNode.Text); tnl > 0 {
				if align := getAlignment(textNode.Text[tnl-1]); align != ast.AlignDefault {
					tn.Align[pos] = align
					textNode.Text = textNode.Text[0 : tnl-1]
				}
			}
		}
	}
}

func (pp *postProcessor) visitTableRows(tn *ast.TableNode, width int) {
	for i, row := range tn.Rows {
		tn.Rows[i] = appendCells(row, width, tn.Align)
		row = tn.Rows[i]
		for i, cell := range row {
			pp.processCell(cell, tn.Align[i])
		}
	}

}

func tableWidth(tn *ast.TableNode) int {
	width := 0
	for _, row := range tn.Rows {
		if width < len(row) {
			width = len(row)
		}
	}
	return width
}

func appendCells(row ast.TableRow, width int, colAlign []ast.Alignment) ast.TableRow {
	for len(row) < width {
		row = append(row, &ast.TableCell{
			Align:   colAlign[len(row)],
			Inlines: &ast.InlineListNode{},
		})
	}
	return row
}

func isHeaderRow(row ast.TableRow) bool {
	for _, cell := range row {
		iln := cell.Inlines
		if inlines := iln.List; len(inlines) > 0 {
			if textNode, ok := inlines[0].(*ast.TextNode); ok {
				if strings.HasPrefix(textNode.Text, "=") {
					return true
				}
			}
		}
	}
	return false
}

func getAlignment(ch byte) ast.Alignment {
	switch ch {
	case ':':
		return ast.AlignCenter
	case '<':
		return ast.AlignLeft
	case '>':
		return ast.AlignRight
	default:
		return ast.AlignDefault
	}
}

// processCell tries to recognize cell formatting.
func (pp *postProcessor) processCell(cell *ast.TableCell, colAlign ast.Alignment) {
	iln := cell.Inlines
	ins := iln.List
	if len(ins) == 0 {
		return
	}
	if textNode, ok := ins[0].(*ast.TextNode); ok && len(textNode.Text) > 0 {
		align := getAlignment(textNode.Text[0])
		if align == ast.AlignDefault {
			cell.Align = colAlign
		} else {
			textNode.Text = textNode.Text[1:]
			cell.Align = align
		}
	} else {
		cell.Align = colAlign
	}
	ast.Walk(pp, iln)
}

var mapSemantic = map[ast.FormatKind]ast.FormatKind{
	ast.FormatItalic: ast.FormatEmph,
	ast.FormatBold:   ast.FormatStrong,
	ast.FormatUnder:  ast.FormatInsert,
	ast.FormatStrike: ast.FormatDelete,
}

func (pp *postProcessor) visitFormat(fn *ast.FormatNode) {
	if fn.Attrs.HasDefault() {
		if newKind, ok := mapSemantic[fn.Kind]; ok {
			fn.Attrs.RemoveDefault()
			fn.Kind = newKind
		}
	}
}

func (pp *postProcessor) visitBlockList(bln *ast.BlockListNode) {
	if bln == nil {
		return
	}
	if len(bln.List) == 0 {
		bln.List = nil
		return
	}
	for _, bn := range bln.List {
		ast.Walk(pp, bn)
	}
	fromPos, toPos := 0, 0
	for fromPos < len(bln.List) {
		bln.List[toPos] = bln.List[fromPos]
		fromPos++
		switch bn := bln.List[toPos].(type) {
		case *ast.ParaNode:
			if len(bn.Inlines.List) > 0 {
				toPos++
			}
		case *nullItemNode:
		case *nullDescriptionNode:
		default:
			toPos++
		}
	}
	for pos := toPos; pos < len(bln.List); pos++ {
		bln.List[pos] = nil // Allow excess nodes to be garbage collected.
	}
	bln.List = bln.List[:toPos:toPos]

}

// processItemSlice post-processes a slice of items.
// It is one of the working horses for post-processing.
func (pp *postProcessor) processItemSlice(ins ast.ItemSlice) ast.ItemSlice {
	if len(ins) == 0 {
		return nil
	}
	for _, in := range ins {
		ast.Walk(pp, in)
	}
	fromPos, toPos := 0, 0
	for fromPos < len(ins) {
		ins[toPos] = ins[fromPos]
		fromPos++
		switch in := ins[toPos].(type) {
		case *ast.ParaNode:
			if in != nil && len(in.Inlines.List) > 0 {
				toPos++
			}
		case *nullItemNode:
		case *nullDescriptionNode:
		default:
			toPos++
		}
	}
	for pos := toPos; pos < len(ins); pos++ {
		ins[pos] = nil // Allow excess nodes to be garbage collected.
	}
	return ins[:toPos:toPos]
}

// processDescriptionSlice post-processes a slice of descriptions.
// It is one of the working horses for post-processing.
func (pp *postProcessor) processDescriptionSlice(dns ast.DescriptionSlice) ast.DescriptionSlice {
	if len(dns) == 0 {
		return nil
	}
	for _, dn := range dns {
		ast.Walk(pp, dn)
	}
	fromPos, toPos := 0, 0
	for fromPos < len(dns) {
		dns[toPos] = dns[fromPos]
		fromPos++
		switch dn := dns[toPos].(type) {
		case *ast.ParaNode:
			if len(dn.Inlines.List) > 0 {
				toPos++
			}
		case *nullDescriptionNode:
		default:
			toPos++
		}
	}
	for pos := toPos; pos < len(dns); pos++ {
		dns[pos] = nil // Allow excess nodes to be garbage collected.
	}
	return dns[:toPos:toPos]
}

func (pp *postProcessor) visitInlineList(iln *ast.InlineListNode) {
	if iln == nil {
		return
	}
	if len(iln.List) == 0 {
		iln.List = nil
		return
	}
	for _, in := range iln.List {
		ast.Walk(pp, in)
	}

	if !pp.inVerse {
		processInlineSliceHead(iln)
	}
	toPos := pp.processInlineSliceCopy(iln)
	toPos = pp.processInlineSliceTail(iln, toPos)
	iln.List = iln.List[:toPos:toPos]
	pp.processInlineListInplace(iln)
}

// processInlineSliceHead removes leading spaces and empty text.
func processInlineSliceHead(iln *ast.InlineListNode) {
	ins := iln.List
	for i, in := range ins {
		switch in := in.(type) {
		case *ast.SpaceNode:
		case *ast.TextNode:
			if len(in.Text) > 0 {
				iln.List = ins[i:]
				return
			}
		default:
			iln.List = ins[i:]
			return
		}
	}
	iln.List = ins[0:0]
}

// processInlineSliceCopy goes forward through the slice and tries to eliminate
// elements that follow the current element.
//
// Two text nodes are merged into one.
//
// Two spaces following a break are merged into a hard break.
func (pp *postProcessor) processInlineSliceCopy(iln *ast.InlineListNode) int {
	ins := iln.List
	maxPos := len(ins)
	for {
		again, toPos := pp.processInlineSliceCopyLoop(iln, maxPos)
		for pos := toPos; pos < maxPos; pos++ {
			ins[pos] = nil // Allow excess nodes to be garbage collected.
		}
		if !again {
			return toPos
		}
		maxPos = toPos
	}
}

func (pp *postProcessor) processInlineSliceCopyLoop(iln *ast.InlineListNode, maxPos int) (bool, int) {
	ins := iln.List
	again := false
	fromPos, toPos := 0, 0
	for fromPos < maxPos {
		ins[toPos] = ins[fromPos]
		fromPos++
		switch in := ins[toPos].(type) {
		case *ast.TextNode:
			for fromPos < maxPos {
				if tn, ok := ins[fromPos].(*ast.TextNode); ok {
					in.Text = in.Text + tn.Text
					fromPos++
				} else {
					break
				}
			}
		case *ast.SpaceNode:
			if fromPos < maxPos {
				switch nn := ins[fromPos].(type) {
				case *ast.BreakNode:
					if len(in.Lexeme) > 1 {
						nn.Hard = true
						ins[toPos] = nn
						fromPos++
					}
				case *ast.TextNode:
					if pp.inVerse {
						ins[toPos] = &ast.TextNode{Text: strings.Repeat("\u00a0", len(in.Lexeme)) + nn.Text}
						fromPos++
						again = true
					}
				}
			}
		case *ast.BreakNode:
			if pp.inVerse {
				in.Hard = true
			}
		}
		toPos++
	}
	return again, toPos
}

// processInlineSliceTail removes empty text nodes, breaks and spaces at the end.
func (pp *postProcessor) processInlineSliceTail(iln *ast.InlineListNode, toPos int) int {
	ins := iln.List
	for toPos > 0 {
		switch n := ins[toPos-1].(type) {
		case *ast.TextNode:
			if len(n.Text) > 0 {
				return toPos
			}
		case *ast.BreakNode:
		case *ast.SpaceNode:
		default:
			return toPos
		}
		toPos--
		ins[toPos] = nil // Kill node to enable garbage collection
	}
	return toPos
}

func (pp *postProcessor) processInlineListInplace(iln *ast.InlineListNode) {
	for _, in := range iln.List {
		if n, ok := in.(*ast.TextNode); ok {
			if n.Text == "..." {
				n.Text = "\u2026"
			} else if len(n.Text) == 4 && strings.IndexByte(",;:!?", n.Text[3]) >= 0 && n.Text[:3] == "..." {
				n.Text = "\u2026" + n.Text[3:]
			}
		}
	}
}
