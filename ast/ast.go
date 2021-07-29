//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package ast provides the abstract syntax tree.
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

// ItemNode is a node that can occur as a list item.
type ItemNode interface {
	BlockNode
	itemNode()
}

// ItemSlice is a slice of ItemNodes.
type ItemSlice []ItemNode

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

// InlineSlice is a slice of InlineNodes.
type InlineSlice []InlineNode

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
	RefStateFound                    // Reference to an existing internal zettel
	RefStateBroken                   // Reference to a non-existing internal zettel
	RefStateHosted                   // Reference to local hosted non-Zettel, without URL change
	RefStateBased                    // Reference to local non-Zettel, to be prefixed
	RefStateExternal                 // Reference to external material
)
