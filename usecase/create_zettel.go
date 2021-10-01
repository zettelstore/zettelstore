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
	"context"

	"zettelstore.de/c/api"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
)

// CreateZettelPort is the interface used by this use case.
type CreateZettelPort interface {
	// CreateZettel creates a new zettel.
	CreateZettel(ctx context.Context, zettel domain.Zettel) (id.Zid, error)
}

// CreateZettel is the data for this use case.
type CreateZettel struct {
	rtConfig config.Config
	port     CreateZettelPort
}

// NewCreateZettel creates a new use case.
func NewCreateZettel(rtConfig config.Config, port CreateZettelPort) CreateZettel {
	return CreateZettel{
		rtConfig: rtConfig,
		port:     port,
	}
}

// Run executes the use case.
func (uc CreateZettel) Run(ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	m := zettel.Meta
	if m.Zid.IsValid() {
		return m.Zid, nil // TODO: new error: already exists
	}

	if title, ok := m.Get(api.KeyTitle); !ok || title == "" {
		m.Set(api.KeyTitle, uc.rtConfig.GetDefaultTitle())
	}
	if role, ok := m.Get(api.KeyRole); !ok || role == "" {
		m.Set(api.KeyRole, uc.rtConfig.GetDefaultRole())
	}
	if syntax, ok := m.Get(api.KeySyntax); !ok || syntax == "" {
		m.Set(api.KeySyntax, uc.rtConfig.GetDefaultSyntax())
	}
	m.YamlSep = uc.rtConfig.GetYAMLHeader()

	zettel.Content.TrimSpace()
	return uc.port.CreateZettel(ctx, zettel)
}
