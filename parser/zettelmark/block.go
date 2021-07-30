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
	"fmt"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/input"
)

// parseBlockSlice parses a sequence of blocks.
func (cp *zmkP) parseBlockSlice() ast.BlockSlice {
	inp := cp.inp
	var lastPara *ast.ParaNode
	result := make(ast.BlockSlice, 0, 2)
	for inp.Ch != input.EOS {
		bn, cont := cp.parseBlock(lastPara)
		if bn != nil {
			result = append(result, bn)
		}
		if !cont {
			lastPara, _ = bn.(*ast.ParaNode)
		}
	}
	if cp.nestingLevel != 0 {
		panic("Nesting level was not decremented")
	}
	return result
}

// parseBlock parses one block.
func (cp *zmkP) parseBlock(lastPara *ast.ParaNode) (res ast.BlockNode, cont bool) {
	inp := cp.inp
	pos := inp.Pos
	if cp.nestingLevel <= maxNestingLevel {
		cp.nestingLevel++
		defer func() { cp.nestingLevel-- }()

		var bn ast.BlockNode
		success := false

		switch inp.Ch {
		case input.EOS:
			return nil, false
		case '\n', '\r':
			inp.EatEOL()
			cp.cleanupListsAfterEOL()
			return nil, false
		case ':':
			bn, success = cp.parseColon()
		case '`', runeModGrave, '%':
			cp.clearStacked()
			bn, success = cp.parseVerbatim()
		case '"', '<':
			cp.clearStacked()
			bn, success = cp.parseRegion()
		case '=':
			cp.clearStacked()
			bn, success = cp.parseHeading()
		case '-':
			cp.clearStacked()
			bn, success = cp.parseHRule()
		case '*', '#', '>':
			cp.table = nil
			cp.descrl = nil
			bn, success = cp.parseNestedList()
		case ';':
			cp.lists = nil
			cp.table = nil
			bn, success = cp.parseDefTerm()
		case ' ':
			cp.table = nil
			bn, success = cp.parseIndent()
		case '|':
			cp.lists = nil
			cp.descrl = nil
			bn, success = cp.parseRow()
		}

		if success {
			return bn, false
		}
	}
	inp.SetPos(pos)
	cp.clearStacked()
	pn := cp.parsePara()
	if lastPara != nil {
		lastPara.Inlines.List = append(lastPara.Inlines.List, pn.Inlines.List...)
		return nil, true
	}
	return pn, false
}

func (cp *zmkP) cleanupListsAfterEOL() {
	for _, l := range cp.lists {
		if lits := len(l.Items); lits > 0 {
			l.Items[lits-1] = append(l.Items[lits-1], &nullItemNode{})
		}
	}
	if cp.descrl != nil {
		defPos := len(cp.descrl.Descriptions) - 1
		if ldds := len(cp.descrl.Descriptions[defPos].Descriptions); ldds > 0 {
			cp.descrl.Descriptions[defPos].Descriptions[ldds-1] = append(
				cp.descrl.Descriptions[defPos].Descriptions[ldds-1], &nullDescriptionNode{})
		}
	}
}

// parseColon determines which element should be parsed.
func (cp *zmkP) parseColon() (ast.BlockNode, bool) {
	inp := cp.inp
	if inp.PeekN(1) == ':' {
		cp.clearStacked()
		return cp.parseRegion()
	}
	return cp.parseDefDescr()
}

// parsePara parses paragraphed inline material.
func (cp *zmkP) parsePara() *ast.ParaNode {
	pn := ast.NewParaNode()
	for {
		in := cp.parseInline()
		if in == nil {
			return pn
		}
		pn.Inlines.List = append(pn.Inlines.List, in)
		if _, ok := in.(*ast.BreakNode); ok {
			ch := cp.inp.Ch
			switch ch {
			// Must contain all cases from above switch in parseBlock.
			case input.EOS, '\n', '\r', '`', runeModGrave, '%', '"', '<', '=', '-', '*', '#', '>', ';', ':', ' ', '|':
				return pn
			}
		}
	}
}

// countDelim read from input until a non-delimiter is found and returns number of delimiter chars.
func (cp *zmkP) countDelim(delim rune) int {
	cnt := 0
	for cp.inp.Ch == delim {
		cnt++
		cp.inp.Next()
	}
	return cnt
}

// parseVerbatim parses a verbatim block.
func (cp *zmkP) parseVerbatim() (rn *ast.VerbatimNode, success bool) {
	inp := cp.inp
	fch := inp.Ch
	cnt := cp.countDelim(fch)
	if cnt < 3 {
		return nil, false
	}
	attrs := cp.parseAttributes(true)
	inp.SkipToEOL()
	if inp.Ch == input.EOS {
		return nil, false
	}
	var kind ast.VerbatimKind
	switch fch {
	case '`', runeModGrave:
		kind = ast.VerbatimProg
	case '%':
		kind = ast.VerbatimComment
	default:
		panic(fmt.Sprintf("%q is not a verbatim char", fch))
	}
	rn = &ast.VerbatimNode{Kind: kind, Attrs: attrs}
	for {
		inp.EatEOL()
		posL := inp.Pos
		switch inp.Ch {
		case fch:
			if cp.countDelim(fch) >= cnt {
				inp.SkipToEOL()
				return rn, true
			}
			inp.SetPos(posL)
		case input.EOS:
			return nil, false
		}
		inp.SkipToEOL()
		rn.Lines = append(rn.Lines, inp.Src[posL:inp.Pos])
	}
}

var runeRegion = map[rune]ast.RegionKind{
	':': ast.RegionSpan,
	'<': ast.RegionQuote,
	'"': ast.RegionVerse,
}

// parseRegion parses a block region.
func (cp *zmkP) parseRegion() (rn *ast.RegionNode, success bool) {
	inp := cp.inp
	fch := inp.Ch
	kind, ok := runeRegion[fch]
	if !ok {
		panic(fmt.Sprintf("%q is not a region char", fch))
	}
	cnt := cp.countDelim(fch)
	if cnt < 3 {
		return nil, false
	}
	attrs := cp.parseAttributes(true)
	inp.SkipToEOL()
	if inp.Ch == input.EOS {
		return nil, false
	}
	rn = &ast.RegionNode{
		Kind:    kind,
		Attrs:   attrs,
		Blocks:  &ast.BlockListNode{},
		Inlines: nil,
	}
	var lastPara *ast.ParaNode
	inp.EatEOL()
	for {
		posL := inp.Pos
		switch inp.Ch {
		case fch:
			if cp.countDelim(fch) >= cnt {
				cp.parseRegionLastLine(rn)
				return rn, true
			}
			inp.SetPos(posL)
		case input.EOS:
			return nil, false
		}
		bn, cont := cp.parseBlock(lastPara)
		if bn != nil {
			rn.Blocks.List = append(rn.Blocks.List, bn)
		}
		if !cont {
			lastPara, _ = bn.(*ast.ParaNode)
		}
	}
}

func (cp *zmkP) parseRegionLastLine(rn *ast.RegionNode) {
	cp.clearStacked() // remove any lists defined in the region
	cp.skipSpace()
	for {
		switch cp.inp.Ch {
		case input.EOS, '\n', '\r':
			return
		}
		in := cp.parseInline()
		if in == nil {
			return
		}
		if rn.Inlines == nil {
			rn.Inlines = &ast.InlineListNode{List: []ast.InlineNode{in}}
		} else {
			rn.Inlines.List = append(rn.Inlines.List, in)
		}
	}

}

// parseHeading parses a head line.
func (cp *zmkP) parseHeading() (hn *ast.HeadingNode, success bool) {
	inp := cp.inp
	lvl := cp.countDelim(inp.Ch)
	if lvl < 3 {
		return nil, false
	}
	if lvl > 7 {
		lvl = 7
	}
	if inp.Ch != ' ' {
		return nil, false
	}
	inp.Next()
	cp.skipSpace()
	hn = &ast.HeadingNode{Level: lvl - 1, Inlines: &ast.InlineListNode{}}
	for {
		if input.IsEOLEOS(inp.Ch) {
			return hn, true
		}
		in := cp.parseInline()
		if in == nil {
			return hn, true
		}
		if inp.Ch == '{' {
			attrs := cp.parseAttributes(true)
			hn.Attrs = attrs
			inp.SkipToEOL()
			return hn, true
		}
		hn.Inlines.List = append(hn.Inlines.List, in)
	}
}

// parseHRule parses a horizontal rule.
func (cp *zmkP) parseHRule() (hn *ast.HRuleNode, success bool) {
	inp := cp.inp
	if cp.countDelim(inp.Ch) < 3 {
		return nil, false
	}
	attrs := cp.parseAttributes(true)
	inp.SkipToEOL()
	return &ast.HRuleNode{Attrs: attrs}, true
}

var mapRuneNestedList = map[rune]ast.NestedListKind{
	'*': ast.NestedListUnordered,
	'#': ast.NestedListOrdered,
	'>': ast.NestedListQuote,
}

// parseNestedList parses a list.
func (cp *zmkP) parseNestedList() (res ast.BlockNode, success bool) {
	inp := cp.inp
	kinds := cp.parseNestedListKinds()
	if kinds == nil {
		return nil, false
	}
	cp.skipSpace()
	if kinds[len(kinds)-1] != ast.NestedListQuote && input.IsEOLEOS(inp.Ch) {
		return nil, false
	}

	if len(kinds) < len(cp.lists) {
		cp.lists = cp.lists[:len(kinds)]
	}
	ln, newLnCount := cp.buildNestedList(kinds)
	pn := cp.parseLinePara()
	if pn == nil {
		pn = ast.NewParaNode()
	}
	ln.Items = append(ln.Items, ast.ItemSlice{pn})
	return cp.cleanupParsedNestedList(newLnCount)
}

func (cp *zmkP) parseNestedListKinds() []ast.NestedListKind {
	inp := cp.inp
	codes := make([]ast.NestedListKind, 0, 4)
	for {
		code, ok := mapRuneNestedList[inp.Ch]
		if !ok {
			panic(fmt.Sprintf("%q is not a region char", inp.Ch))
		}
		codes = append(codes, code)
		inp.Next()
		switch inp.Ch {
		case '*', '#', '>':
		case ' ', input.EOS, '\n', '\r':
			return codes
		default:
			return nil
		}
	}

}

func (cp *zmkP) buildNestedList(kinds []ast.NestedListKind) (ln *ast.NestedListNode, newLnCount int) {
	for i, kind := range kinds {
		if i < len(cp.lists) {
			if cp.lists[i].Kind != kind {
				ln = &ast.NestedListNode{Kind: kind}
				newLnCount++
				cp.lists[i] = ln
				cp.lists = cp.lists[:i+1]
			} else {
				ln = cp.lists[i]
			}
		} else {
			ln = &ast.NestedListNode{Kind: kind}
			newLnCount++
			cp.lists = append(cp.lists, ln)
		}
	}
	return ln, newLnCount
}

func (cp *zmkP) cleanupParsedNestedList(newLnCount int) (res ast.BlockNode, success bool) {
	listDepth := len(cp.lists)
	for i := 0; i < newLnCount; i++ {
		childPos := listDepth - i - 1
		parentPos := childPos - 1
		if parentPos < 0 {
			return cp.lists[0], true
		}
		if prevItems := cp.lists[parentPos].Items; len(prevItems) > 0 {
			lastItem := len(prevItems) - 1
			prevItems[lastItem] = append(prevItems[lastItem], cp.lists[childPos])
		} else {
			cp.lists[parentPos].Items = []ast.ItemSlice{{cp.lists[childPos]}}
		}
	}
	return nil, true

}

// parseDefTerm parses a term of a definition list.
func (cp *zmkP) parseDefTerm() (res ast.BlockNode, success bool) {
	inp := cp.inp
	inp.Next()
	if inp.Ch != ' ' {
		return nil, false
	}
	inp.Next()
	cp.skipSpace()
	descrl := cp.descrl
	if descrl == nil {
		descrl = &ast.DescriptionListNode{}
		cp.descrl = descrl
	}
	descrl.Descriptions = append(descrl.Descriptions, ast.Description{})
	defPos := len(descrl.Descriptions) - 1
	if defPos == 0 {
		res = descrl
	}
	for {
		in := cp.parseInline()
		if in == nil {
			if descrl.Descriptions[defPos].Term == nil {
				return nil, false
			}
			return res, true
		}
		if descrl.Descriptions[defPos].Term == nil {
			descrl.Descriptions[defPos].Term = &ast.InlineListNode{}
		}
		descrl.Descriptions[defPos].Term.List = append(descrl.Descriptions[defPos].Term.List, in)
		if _, ok := in.(*ast.BreakNode); ok {
			return res, true
		}
	}
}

// parseDefDescr parses a description of a definition list.
func (cp *zmkP) parseDefDescr() (res ast.BlockNode, success bool) {
	inp := cp.inp
	inp.Next()
	if inp.Ch != ' ' {
		return nil, false
	}
	inp.Next()
	cp.skipSpace()
	descrl := cp.descrl
	if descrl == nil || len(descrl.Descriptions) == 0 {
		return nil, false
	}
	defPos := len(descrl.Descriptions) - 1
	if descrl.Descriptions[defPos].Term == nil {
		return nil, false
	}
	pn := cp.parseLinePara()
	if pn == nil {
		return nil, false
	}
	cp.lists = nil
	cp.table = nil
	descrl.Descriptions[defPos].Descriptions = append(descrl.Descriptions[defPos].Descriptions, ast.DescriptionSlice{pn})
	return nil, true
}

// parseIndent parses initial spaces to continue a list.
func (cp *zmkP) parseIndent() (res ast.BlockNode, success bool) {
	inp := cp.inp
	cnt := 0
	for {
		inp.Next()
		if inp.Ch != ' ' {
			break
		}
		cnt++
	}
	if cp.lists != nil {
		return nil, cp.parseIndentForList(cnt)
	}
	if cp.descrl != nil {
		return nil, cp.parseIndentForDescription(cnt)
	}
	return nil, false
}

func (cp *zmkP) parseIndentForList(cnt int) bool {
	if len(cp.lists) < cnt {
		cnt = len(cp.lists)
	}
	cp.lists = cp.lists[:cnt]
	if cnt == 0 {
		return false
	}
	ln := cp.lists[cnt-1]
	pn := cp.parseLinePara()
	if pn == nil {
		pn = ast.NewParaNode()
	}
	lbn := ln.Items[len(ln.Items)-1]
	if lpn, ok := lbn[len(lbn)-1].(*ast.ParaNode); ok {
		lpn.Inlines.List = append(lpn.Inlines.List, pn.Inlines.List...)
	} else {
		ln.Items[len(ln.Items)-1] = append(ln.Items[len(ln.Items)-1], pn)
	}
	return true
}

func (cp *zmkP) parseIndentForDescription(cnt int) bool {
	defPos := len(cp.descrl.Descriptions) - 1
	if cnt < 1 || defPos < 0 {
		return false
	}
	if len(cp.descrl.Descriptions[defPos].Descriptions) == 0 {
		// Continuation of a definition term
		for {
			in := cp.parseInline()
			if in == nil {
				return true
			}
			cp.descrl.Descriptions[defPos].Term.List = append(cp.descrl.Descriptions[defPos].Term.List, in)
			if _, ok := in.(*ast.BreakNode); ok {
				return true
			}
		}
	}

	// Continuation of a definition description
	pn := cp.parseLinePara()
	if pn == nil {
		return false
	}
	descrPos := len(cp.descrl.Descriptions[defPos].Descriptions) - 1
	lbn := cp.descrl.Descriptions[defPos].Descriptions[descrPos]
	if lpn, ok := lbn[len(lbn)-1].(*ast.ParaNode); ok {
		lpn.Inlines.List = append(lpn.Inlines.List, pn.Inlines.List...)
	} else {
		descrPos := len(cp.descrl.Descriptions[defPos].Descriptions) - 1
		cp.descrl.Descriptions[defPos].Descriptions[descrPos] = append(cp.descrl.Descriptions[defPos].Descriptions[descrPos], pn)
	}
	return true
}

// parseLinePara parses one line of inline material.
func (cp *zmkP) parseLinePara() *ast.ParaNode {
	pn := ast.NewParaNode()
	for {
		in := cp.parseInline()
		if in == nil {
			if pn.Inlines == nil {
				return nil
			}
			return pn
		}
		pn.Inlines.List = append(pn.Inlines.List, in)
		if _, ok := in.(*ast.BreakNode); ok {
			return pn
		}
	}
}

// parseRow parse one table row.
func (cp *zmkP) parseRow() (res ast.BlockNode, success bool) {
	inp := cp.inp
	if inp.Peek() == '%' {
		inp.SkipToEOL()
		return nil, true
	}
	row := ast.TableRow{}
	for {
		inp.Next()
		cell := cp.parseCell()
		if cell != nil {
			row = append(row, cell)
		}
		switch inp.Ch {
		case '\n', '\r':
			inp.EatEOL()
			fallthrough
		case input.EOS:
			// add to table
			if cp.table == nil {
				cp.table = &ast.TableNode{Rows: []ast.TableRow{row}}
				return cp.table, true
			}
			cp.table.Rows = append(cp.table.Rows, row)
			return nil, true
		}
		// inp.Ch must be '|'
	}
}

// parseCell parses one single cell of a table row.
func (cp *zmkP) parseCell() *ast.TableCell {
	inp := cp.inp
	var l []ast.InlineNode
	for {
		if input.IsEOLEOS(inp.Ch) {
			if len(l) == 0 {
				return nil
			}
			return &ast.TableCell{Inlines: &ast.InlineListNode{List: l}}
		}
		if inp.Ch == '|' {
			return &ast.TableCell{Inlines: &ast.InlineListNode{List: l}}
		}
		l = append(l, cp.parseInline())
	}
}
