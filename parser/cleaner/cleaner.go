//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package cleaner provides funxtions to clean up the parsed AST.
package cleaner

import (
	"strconv"
	"strings"

	"zettelstore.de/z/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/strfun"
)

// CleanBlockList cleans the given block list.
func CleanBlockList(bln *ast.BlockListNode) { cleanNode(bln) }

// CleanInlineList cleans the given inline list.
func CleanInlineList(iln *ast.InlineListNode) { cleanNode(iln) }

func cleanNode(n ast.Node) {
	cv := cleanVisitor{
		textEnc: encoder.Create(api.EncoderText, nil),
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
	textEnc encoder.Encoder
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
	if cv.doMark || hn == nil || hn.Inlines.IsEmpty() {
		return
	}
	if hn.Slug == "" {
		var sb strings.Builder
		_, err := cv.textEnc.WriteInlines(&sb, hn.Inlines)
		if err != nil {
			return
		}
		hn.Slug = strfun.Slugify(sb.String())
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
	if mn.Text == "" {
		mn.Slug = ""
		mn.Fragment = cv.addIdentifier("*", mn)
		return
	}
	if mn.Slug == "" {
		mn.Slug = strfun.Slugify(mn.Text)
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
			if n, ok := cv.ids[newID]; !ok || n == node {
				cv.ids[newID] = node
				return newID
			}
		}
	}
	cv.ids[id] = node
	return id
}
