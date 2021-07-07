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
	"zettelstore.de/z/domain/meta"
)

// CopyZettel is the data for this use case.
type CopyZettel struct{}

// NewCopyZettel creates a new use case.
func NewCopyZettel() CopyZettel {
	return CopyZettel{}
}

// Run executes the use case.
func (uc CopyZettel) Run(origZettel domain.Zettel) domain.Zettel {
	m := origZettel.Meta.Clone()
	if title, ok := m.Get(meta.KeyTitle); ok {
		if len(title) > 0 {
			title = "Copy of " + title
		} else {
			title = "Copy"
		}
		m.Set(meta.KeyTitle, title)
	}
	content := origZettel.Content
	content.TrimSpace()
	return domain.Zettel{Meta: m, Content: content}
}
