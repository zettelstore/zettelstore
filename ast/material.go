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

// MaterialNode references the various types of zettel material.
type MaterialNode interface {
	Node
	materialNode()
}

// --------------------------------------------------------------------------

// ReferenceMaterialNode is material that can be retrieved by using a reference.
type ReferenceMaterialNode struct {
	Ref *Reference
}

func (*ReferenceMaterialNode) materialNode() { /* Just a marker */ }

// WalkChildren does nothing.
func (*ReferenceMaterialNode) WalkChildren(Visitor) { /* No children*/ }

// --------------------------------------------------------------------------

// BLOBMaterialNode represents itself.
type BLOBMaterialNode struct {
	Blob   []byte // BLOB data itself.
	Syntax string // Syntax of Blob
}

func (*BLOBMaterialNode) materialNode() { /* Just a marker */ }

// WalkChildren does nothing.
func (*BLOBMaterialNode) WalkChildren(Visitor) { /* No children*/ }
