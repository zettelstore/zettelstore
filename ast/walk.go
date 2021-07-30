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

// Visitor is a visitor for walking the AST.
type Visitor interface {
	Visit(node Node) Visitor
}

// Walk traverses the AST.
func Walk(v Visitor, node Node) {
	if v = v.Visit(node); v == nil {
		return
	}
	node.WalkChildren(v)
	v.Visit(nil)
}

// WalkItemSlice traverses an item slice.
func WalkItemSlice(v Visitor, ins ItemSlice) {
	for _, in := range ins {
		Walk(v, in)
	}
}

// WalkDescriptionSlice traverses an item slice.
func WalkDescriptionSlice(v Visitor, dns DescriptionSlice) {
	for _, dn := range dns {
		Walk(v, dn)
	}
}
