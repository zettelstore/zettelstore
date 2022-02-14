//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package ast provides the abstract syntax tree for parsed zettel content.
package ast

import (
	"net/url"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

// ZettelNode is the root node of the abstract syntax tree.
// It is *not* part of the visitor pattern.
type ZettelNode struct {
	Meta    *meta.Meta     // Original metadata
	Content domain.Content // Original content
	Zid     id.Zid         // Zettel identification.
	InhMeta *meta.Meta     // Metadata of the zettel, with inherited values.
	Ast     BlockSlice     // Zettel abstract syntax tree is a sequence of block nodes.
	Syntax  string         // Syntax / parser that produced the Ast
}

// Node is the interface, all nodes must implement.
type Node interface {
	WalkChildren(v Visitor)
}

// BlockNode is the interface that all block nodes must implement.
type BlockNode interface {
	Node
	blockNode()
}

// BlockSlice is a slice of BlockNodes.
type BlockSlice []BlockNode

func (*BlockSlice) blockNode() { /* Just a marker */ }

// WalkChildren walks down to the descriptions.
func (bs *BlockSlice) WalkChildren(v Visitor) {
	if bs != nil {
		for _, bn := range *bs {
			Walk(v, bn)
		}
	}
}

// FirstParagraphInlines returns the inline list of the first paragraph that
// contains a inline list.
func (bs BlockSlice) FirstParagraphInlines() InlineSlice {
	if len(bs) > 0 {
		for _, bn := range bs {
			pn, ok := bn.(*ParaNode)
			if !ok {
				continue
			}
			if inl := pn.Inlines; len(inl) > 0 {
				return inl
			}
		}
	}
	return nil
}

// ItemNode is a node that can occur as a list item.
type ItemNode interface {
	BlockNode
	itemNode()
}

// ItemSlice is a slice of ItemNodes.
type ItemSlice []ItemNode

// ItemListNode is a list of BlockNodes.
type ItemListNode struct {
	List []ItemNode
}

// WalkChildren walks down to the descriptions.
func (iln *ItemListNode) WalkChildren(v Visitor) {
	for _, bn := range iln.List {
		Walk(v, bn)
	}
}

// DescriptionNode is a node that contains just textual description.
type DescriptionNode interface {
	ItemNode
	descriptionNode()
}

// DescriptionSlice is a slice of DescriptionNodes.
type DescriptionSlice []DescriptionNode

// InlineNode is the interface that all inline nodes must implement.
type InlineNode interface {
	Node
	inlineNode()
}

// InlineEmbedNode is a node that specifies some embeddings in inline mode.
// It is abstract, b/c there are different concrete type implementations.
type InlineEmbedNode interface {
	InlineNode
	inlineEmbedNode()
}

// Reference is a reference to external or internal material.
type Reference struct {
	URL   *url.URL
	Value string
	State RefState
}

// RefState indicates the state of the reference.
type RefState int

// Constants for RefState
const (
	RefStateInvalid  RefState = iota // Invalid Reference
	RefStateZettel                   // Reference to an internal zettel
	RefStateSelf                     // Reference to same zettel with a fragment
	RefStateFound                    // Reference to an existing internal zettel, URL is ajusted
	RefStateBroken                   // Reference to a non-existing internal zettel
	RefStateHosted                   // Reference to local hosted non-Zettel, without URL change
	RefStateBased                    // Reference to local non-Zettel, to be prefixed
	RefStateExternal                 // Reference to external material
)
