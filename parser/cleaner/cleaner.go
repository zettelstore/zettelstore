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

	"zettelstore.de/z/ast"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/strfun"
)

// CleanupBlockSlice cleans the given block slice.
func CleanupBlockSlice(bs ast.BlockSlice) {
	cv := cleanupVisitor{
		textEnc: encoder.Create("text", nil),
		hasMark: false,
		doMark:  false,
	}
	ast.WalkBlockSlice(&cv, bs)
	if cv.hasMark {
		cv.doMark = true
		ast.WalkBlockSlice(&cv, bs)
	}
}

type cleanupVisitor struct {
	textEnc encoder.Encoder
	ids     map[string]ast.Node
	hasMark bool
	doMark  bool
}

func (cv *cleanupVisitor) Visit(node ast.Node) ast.WalkVisitor {
	switch n := node.(type) {
	case *ast.HeadingNode:
		if cv.doMark || n == nil || n.Inlines == nil {
			return nil
		}
		var sb strings.Builder
		_, err := cv.textEnc.WriteInlines(&sb, n.Inlines)
		if err != nil {
			return nil
		}
		s := strfun.Slugify(sb.String())
		if len(s) > 0 {
			n.Slug = cv.addIdentifier(s, n)
		}
		return nil
	case *ast.MarkNode:
		if !cv.doMark {
			cv.hasMark = true
			return nil
		}
		if n.Text == "" {
			n.Text = cv.addIdentifier("*", n)
			return nil
		}
		n.Text = cv.addIdentifier(n.Text, n)
		return nil
	}
	return cv
}

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
