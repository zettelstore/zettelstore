//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package ast provides the abstract syntax tree.
package ast

// WalkVisitor is a visitor for walking the AST.
type WalkVisitor interface {
	Visit(node Node) WalkVisitor
}

// Walk traverses the AST.
func Walk(v WalkVisitor, node Node) {
	if v = v.Visit(node); v == nil {
		return
	}
	node.WalkChildren(v)
	v.Visit(nil)
}

// WalkBlockSlice traverse a block slice.
func WalkBlockSlice(v WalkVisitor, bns BlockSlice) {
	for _, bn := range bns {
		Walk(v, bn)
	}
}

// WalkInlineSlice traverses an inline slice.
func WalkInlineSlice(v WalkVisitor, ins InlineSlice) {
	for _, in := range ins {
		Walk(v, in)
	}
}

// WalkItemSlice traverses an item slice.
func WalkItemSlice(v WalkVisitor, ins ItemSlice) {
	for _, in := range ins {
		Walk(v, in)
	}
}

// WalkDescriptionSlice traverses an item slice.
func WalkDescriptionSlice(v WalkVisitor, dns DescriptionSlice) {
	for _, dn := range dns {
		Walk(v, dn)
	}
}
