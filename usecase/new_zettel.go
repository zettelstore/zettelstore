//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package usecase provides (business) use cases for the zettelstore.
package usecase

import (
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

// NewZettel is the data for this use case.
type NewZettel struct{}

// NewNewZettel creates a new use case.
func NewNewZettel() NewZettel {
	return NewZettel{}
}

// Run executes the use case.
func (uc NewZettel) Run(origZettel domain.Zettel) domain.Zettel {
	m := meta.New(id.Invalid)
	om := origZettel.Meta
	m.Set(meta.KeyTitle, om.GetDefault(meta.KeyTitle, ""))
	m.Set(meta.KeyRole, om.GetDefault(meta.KeyRole, ""))
	m.Set(meta.KeyTags, om.GetDefault(meta.KeyTags, ""))
	m.Set(meta.KeySyntax, om.GetDefault(meta.KeySyntax, ""))

	const prefixLen = len(meta.NewPrefix)
	for _, pair := range om.PairsRest(false) {
		if key := pair.Key; len(key) > prefixLen && key[0:prefixLen] == meta.NewPrefix {
			m.Set(key[prefixLen:], pair.Value)
		}
	}
	content := origZettel.Content
	content.TrimSpace()
	return domain.Zettel{Meta: m, Content: content}
}
