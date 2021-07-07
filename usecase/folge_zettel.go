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
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

// FolgeZettel is the data for this use case.
type FolgeZettel struct {
	rtConfig config.Config
}

// NewFolgeZettel creates a new use case.
func NewFolgeZettel(rtConfig config.Config) FolgeZettel {
	return FolgeZettel{rtConfig}
}

// Run executes the use case.
func (uc FolgeZettel) Run(origZettel domain.Zettel) domain.Zettel {
	origMeta := origZettel.Meta
	m := meta.New(id.Invalid)
	if title, ok := origMeta.Get(meta.KeyTitle); ok {
		if len(title) > 0 {
			title = "Folge of " + title
		} else {
			title = "Folge"
		}
		m.Set(meta.KeyTitle, title)
	}
	m.Set(meta.KeyRole, config.GetRole(origMeta, uc.rtConfig))
	m.Set(meta.KeyTags, origMeta.GetDefault(meta.KeyTags, ""))
	m.Set(meta.KeySyntax, uc.rtConfig.GetDefaultSyntax())
	m.Set(meta.KeyPrecursor, origMeta.Zid.String())
	return domain.Zettel{Meta: m, Content: domain.NewContent("")}
}
