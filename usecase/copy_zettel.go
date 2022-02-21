//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package usecase

import (
	"zettelstore.de/c/api"
	"zettelstore.de/z/domain"
)

// CopyZettel is the data for this use case.
type CopyZettel struct{}

// NewCopyZettel creates a new use case.
func NewCopyZettel() CopyZettel {
	return CopyZettel{}
}

// Run executes the use case.
func (CopyZettel) Run(origZettel domain.Zettel) domain.Zettel {
	m := origZettel.Meta.Clone()
	if title, ok := m.Get(api.KeyTitle); ok {
		m.Set(api.KeyTitle, copyTitle(title))
	}
	if readonly, ok := m.Get(api.KeyReadOnly); ok {
		m.Set(api.KeyReadOnly, copyReadonly(readonly))
	}
	content := origZettel.Content
	content.TrimSpace()
	return domain.Zettel{Meta: m, Content: content}
}

func copyTitle(title string) string {
	if len(title) > 0 {
		return "Copy of " + title
	}
	return "Copy"
}

func copyReadonly(s string) string {
	// Currently, "false" is a safe value.
	//
	// If the current user and its role is known, a mor elaborative calculation
	// could be done: set it to a value, so that the current user will be able
	// to modify it later.
	return api.ValueFalse
}
