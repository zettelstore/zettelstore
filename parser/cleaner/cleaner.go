//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package cleaner provides funxtions to clean up the parsed AST.
package cleaner

import (
	"bytes"
	"strconv"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/encoder/textenc"
	"zettelstore.de/z/strfun"
)

// CleanBlockSlice cleans the given block list.
func CleanBlockSlice(bs *ast.BlockSlice) { cleanNode(bs) }

// CleanInlineSlice cleans the given inline list.
func CleanInlineSlice(is *ast.InlineSlice) { cleanNode(is) }

func cleanNode(n ast.Node) {
	cv := cleanVisitor{
		textEnc: textenc.Create(),
		hasMark: false,
		doMark:  false,
	}
	ast.Walk(&cv, n)
	if cv.hasMark {
		cv.doMark = true
		ast.Walk(&cv, n)
	}
}

type cleanVisitor struct {
	textEnc *textenc.Encoder
	ids     map[string]ast.Node
	hasMark bool
	doMark  bool
}

func (cv *cleanVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.HeadingNode:
		cv.visitHeading(n)
		return nil
	case *ast.MarkNode:
		cv.visitMark(n)
		return nil
	}
	return cv
}

func (cv *cleanVisitor) visitHeading(hn *ast.HeadingNode) {
	if cv.doMark || hn == nil || len(hn.Inlines) == 0 {
		return
	}
	if hn.Slug == "" {
		var buf bytes.Buffer
		_, err := cv.textEnc.WriteInlines(&buf, &hn.Inlines)
		if err != nil {
			return
		}
		hn.Slug = strfun.Slugify(buf.String())
	}
	if hn.Slug != "" {
		hn.Fragment = cv.addIdentifier(hn.Slug, hn)
	}
}

func (cv *cleanVisitor) visitMark(mn *ast.MarkNode) {
	if !cv.doMark {
		cv.hasMark = true
		return
	}
	if mn.Mark == "" {
		mn.Slug = ""
		mn.Fragment = cv.addIdentifier("*", mn)
		return
	}
	if mn.Slug == "" {
		mn.Slug = strfun.Slugify(mn.Mark)
	}
	mn.Fragment = cv.addIdentifier(mn.Slug, mn)
}

func (cv *cleanVisitor) addIdentifier(id string, node ast.Node) string {
	if cv.ids == nil {
		cv.ids = map[string]ast.Node{id: node}
		return id
	}
	if n, ok := cv.ids[id]; ok && n != node {
		prefix := id + "-"
		for count := 1; ; count++ {
			newID := prefix + strconv.Itoa(count)
			if n2, ok2 := cv.ids[newID]; !ok2 || n2 == node {
				cv.ids[newID] = node
				return newID
			}
		}
	}
	cv.ids[id] = node
	return id
}

// CleanInlineLinks removes all links and footnote node from the given inline slice.
func CleanInlineLinks(is *ast.InlineSlice) { ast.Walk(&cleanLinks{}, is) }

type cleanLinks struct{}

func (cl *cleanLinks) Visit(node ast.Node) ast.Visitor {
	ins, ok := node.(*ast.InlineSlice)
	if !ok {
		return cl
	}
	for _, in := range *ins {
		ast.Walk(cl, in)
	}
	if hasNoLinks(*ins) {
		return nil
	}

	result := make(ast.InlineSlice, 0, len(*ins))
	for _, in := range *ins {
		switch n := in.(type) {
		case *ast.LinkNode:
			result = append(result, n.Inlines...)
		case *ast.FootnoteNode: // Do nothing
		default:
			result = append(result, n)
		}
	}
	*ins = result
	return nil
}

func hasNoLinks(ins ast.InlineSlice) bool {
	for _, in := range ins {
		switch in.(type) {
		case *ast.LinkNode, *ast.FootnoteNode:
			return false
		}
	}
	return true
}
