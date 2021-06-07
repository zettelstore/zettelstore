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
func postProcessBlocks(bs ast.BlockSlice) ast.BlockSlice {
	pp := postProcessor{}
	return pp.processBlockSlice(bs)
}

// postProcessInlines is the entry point for post-processing a list of inline nodes.
func postProcessInlines(is ast.InlineSlice) ast.InlineSlice {
	pp := postProcessor{}
	return pp.processInlineSlice(is)
}

// postProcessor is a visitor that cleans the abstract syntax tree.
type postProcessor struct {
	inVerse bool
}

func (pp *postProcessor) Visit(node ast.Node) ast.WalkVisitor {
	switch n := node.(type) {
	case *ast.ParaNode:
		n.Inlines = pp.processInlineSlice(n.Inlines)
	case *ast.RegionNode:
		oldVerse := pp.inVerse
		if n.Code == ast.RegionVerse {
			pp.inVerse = true
		}
		n.Blocks = pp.processBlockSlice(n.Blocks)
		pp.inVerse = oldVerse
		n.Inlines = pp.processInlineSlice(n.Inlines)
	case *ast.HeadingNode:
		n.Inlines = pp.processInlineSlice(n.Inlines)
	case *ast.NestedListNode:
		for i, item := range n.Items {
			n.Items[i] = pp.processItemSlice(item)
		}
	case *ast.DescriptionListNode:
		for i, def := range n.Descriptions {
			n.Descriptions[i].Term = pp.processInlineSlice(def.Term)
			for j, b := range def.Descriptions {
				n.Descriptions[i].Descriptions[j] = pp.processDescriptionSlice(b)
			}
		}
	case *ast.TableNode:
		width := tableWidth(n)
		n.Align = make([]ast.Alignment, width)
		for i := 0; i < width; i++ {
			n.Align[i] = ast.AlignDefault
		}
		if len(n.Rows) > 0 && isHeaderRow(n.Rows[0]) {
			n.Header = n.Rows[0]
			n.Rows = n.Rows[1:]
			pp.visitTableHeader(n)
		}
		if len(n.Header) > 0 {
			n.Header = appendCells(n.Header, width, n.Align)
			for i, cell := range n.Header {
				pp.processCell(cell, n.Align[i])
			}
		}
		pp.visitTableRows(n, width)
	case *ast.LinkNode:
		n.Inlines = pp.processInlineSlice(n.Inlines)
	case *ast.ImageNode:
		n.Inlines = pp.processInlineSlice(n.Inlines)
	case *ast.CiteNode:
		n.Inlines = pp.processInlineSlice(n.Inlines)
	case *ast.FootnoteNode:
		n.Inlines = pp.processInlineSlice(n.Inlines)
	case *ast.FormatNode:
		if n.Attrs != nil && n.Attrs.HasDefault() {
			if newCode, ok := mapSemantic[n.Code]; ok {
				n.Attrs.RemoveDefault()
				n.Code = newCode
			}
		}
		n.Inlines = pp.processInlineSlice(n.Inlines)
	}
	return nil
}

func (pp *postProcessor) visitTableHeader(tn *ast.TableNode) {
	for pos, cell := range tn.Header {
		inlines := cell.Inlines
		if len(inlines) == 0 {
			continue
		}
		if textNode, ok := inlines[0].(*ast.TextNode); ok {
			textNode.Text = strings.TrimPrefix(textNode.Text, "=")
		}
		if textNode, ok := inlines[len(inlines)-1].(*ast.TextNode); ok {
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
		row = append(row, &ast.TableCell{Align: colAlign[len(row)]})
	}
	return row
}

func isHeaderRow(row ast.TableRow) bool {
	for i := 0; i < len(row); i++ {
		if inlines := row[i].Inlines; len(inlines) > 0 {
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
	if len(cell.Inlines) == 0 {
		return
	}
	if textNode, ok := cell.Inlines[0].(*ast.TextNode); ok && len(textNode.Text) > 0 {
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
	cell.Inlines = pp.processInlineSlice(cell.Inlines)
}

var mapSemantic = map[ast.FormatCode]ast.FormatCode{
	ast.FormatItalic: ast.FormatEmph,
	ast.FormatBold:   ast.FormatStrong,
	ast.FormatUnder:  ast.FormatInsert,
	ast.FormatStrike: ast.FormatDelete,
}

// processBlockSlice post-processes a slice of blocks.
// It is one of the working horses for post-processing.
func (pp *postProcessor) processBlockSlice(bns ast.BlockSlice) ast.BlockSlice {
	if len(bns) == 0 {
		return nil
	}
	ast.WalkBlockSlice(pp, bns)
	fromPos, toPos := 0, 0
	for fromPos < len(bns) {
		bns[toPos] = bns[fromPos]
		fromPos++
		switch bn := bns[toPos].(type) {
		case *ast.ParaNode:
			if len(bn.Inlines) > 0 {
				toPos++
			}
		case *nullItemNode:
		case *nullDescriptionNode:
		default:
			toPos++
		}
	}
	for pos := toPos; pos < len(bns); pos++ {
		bns[pos] = nil // Allow excess nodes to be garbage collected.
	}
	return bns[:toPos:toPos]
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
			if in != nil && len(in.Inlines) > 0 {
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
			if len(dn.Inlines) > 0 {
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

// processInlineSlice post-processes a slice of inline nodes.
// It is one of the working horses for post-processing.
func (pp *postProcessor) processInlineSlice(ins ast.InlineSlice) ast.InlineSlice {
	if len(ins) == 0 {
		return nil
	}
	ast.WalkInlineSlice(pp, ins)

	if !pp.inVerse {
		ins = processInlineSliceHead(ins)
	}
	toPos := pp.processInlineSliceCopy(ins)
	toPos = pp.processInlineSliceTail(ins, toPos)
	ins = ins[:toPos:toPos]
	pp.processInlineSliceInplace(ins)
	return ins
}

// processInlineSliceHead removes leading spaces and empty text.
func processInlineSliceHead(ins ast.InlineSlice) ast.InlineSlice {
	for i := 0; i < len(ins); i++ {
		switch in := ins[i].(type) {
		case *ast.SpaceNode:
		case *ast.TextNode:
			if len(in.Text) > 0 {
				return ins[i:]
			}
		default:
			return ins[i:]
		}
	}
	return ins[0:0]
}

// processInlineSliceCopy goes forward through the slice and tries to eliminate
// elements that follow the current element.
//
// Two text nodes are merged into one.
//
// Two spaces following a break are merged into a hard break.
func (pp *postProcessor) processInlineSliceCopy(ins ast.InlineSlice) int {
	maxPos := len(ins)
	for {
		again, toPos := pp.processInlineSliceCopyLoop(ins, maxPos)
		for pos := toPos; pos < maxPos; pos++ {
			ins[pos] = nil // Allow excess nodes to be garbage collected.
		}
		if !again {
			return toPos
		}
		maxPos = toPos
	}
}

func (pp *postProcessor) processInlineSliceCopyLoop(
	ins ast.InlineSlice, maxPos int) (bool, int) {

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
func (pp *postProcessor) processInlineSliceTail(ins ast.InlineSlice, toPos int) int {
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

func (pp *postProcessor) processInlineSliceInplace(ins ast.InlineSlice) {
	for _, in := range ins {
		if n, ok := in.(*ast.TextNode); ok {
			if n.Text == "..." {
				n.Text = "\u2026"
			} else if len(n.Text) == 4 && strings.IndexByte(",;:!?", n.Text[3]) >= 0 && n.Text[:3] == "..." {
				n.Text = "\u2026" + n.Text[3:]
			}
		}
	}
}
