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
func CleanBlockList(bln *ast.BlockListNode) {
	cv := cleanVisitor{
		textEnc: encoder.Create(api.EncoderText, nil),
		hasMark: false,
		doMark:  false,
	}
	ast.Walk(&cv, bln)
	if cv.hasMark {
		cv.doMark = true
		ast.Walk(&cv, bln)
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
		if cv.doMark || n == nil || n.Inlines == nil {
			return nil
		}
		if n.Slug == "" {
			var sb strings.Builder
			_, err := cv.textEnc.WriteInlines(&sb, n.Inlines)
			if err != nil {
				return nil
			}
			n.Slug = strfun.Slugify(sb.String())
		}
		if n.Slug != "" {
			n.Fragment = cv.addIdentifier(n.Slug, n)
		}
		return nil
	case *ast.MarkNode:
		if !cv.doMark {
			cv.hasMark = true
			return nil
		}
		if n.Text == "" {
			n.Slug = ""
			n.Fragment = cv.addIdentifier("*", n)
			return nil
		}
		if n.Slug == "" {
			n.Slug = strfun.Slugify(n.Text)
		}
		n.Fragment = cv.addIdentifier(n.Slug, n)
		return nil
	}
	return cv
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
