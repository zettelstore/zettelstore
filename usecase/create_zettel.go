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

	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/strfun"
)

// CreateZettelPort is the interface used by this use case.
type CreateZettelPort interface {
	// CreateZettel creates a new zettel.
	CreateZettel(ctx context.Context, zettel domain.Zettel) (id.Zid, error)
}

// CreateZettel is the data for this use case.
type CreateZettel struct {
	port CreateZettelPort
}

// NewCreateZettel creates a new use case.
func NewCreateZettel(port CreateZettelPort) CreateZettel {
	return CreateZettel{port: port}
}

// Run executes the use case.
func (uc CreateZettel) Run(ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	m := zettel.Meta
	if m.Zid.IsValid() {
		return m.Zid, nil // TODO: new error: already exists
	}

	if title, ok := m.Get(meta.KeyTitle); !ok || title == "" {
		m.Set(meta.KeyTitle, runtime.GetDefaultTitle())
	}
	if role, ok := m.Get(meta.KeyRole); !ok || role == "" {
		m.Set(meta.KeyRole, runtime.GetDefaultRole())
	}
	if syntax, ok := m.Get(meta.KeySyntax); !ok || syntax == "" {
		m.Set(meta.KeySyntax, runtime.GetDefaultSyntax())
	}
	m.YamlSep = runtime.GetYAMLHeader()

	zettel.Content = domain.Content(strfun.TrimSpaceRight(zettel.Content.AsString()))
	return uc.port.CreateZettel(ctx, zettel)
}
