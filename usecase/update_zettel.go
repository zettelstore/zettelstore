//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
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

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

// UpdateZettelPort is the interface used by this use case.
type UpdateZettelPort interface {
	// GetZettel retrieves a specific zettel.
	GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error)

	// UpdateZettel updates an existing zettel.
	UpdateZettel(ctx context.Context, zettel domain.Zettel) error
}

// UpdateZettel is the data for this use case.
type UpdateZettel struct {
	port UpdateZettelPort
}

// NewUpdateZettel creates a new use case.
func NewUpdateZettel(port UpdateZettelPort) UpdateZettel {
	return UpdateZettel{port: port}
}

// Run executes the use case.
func (uc UpdateZettel) Run(ctx context.Context, zettel domain.Zettel, hasContent bool) error {
	m := zettel.Meta
	oldZettel, err := uc.port.GetZettel(ctx, m.Zid)
	if err != nil {
		return err
	}
	if zettel.Equal(oldZettel, false) {
		return nil
	}
	m.SetNow(meta.KeyModified)
	m.YamlSep = oldZettel.Meta.YamlSep
	if m.Zid == id.ConfigurationZid {
		m.Set(meta.KeySyntax, meta.ValueSyntaxMeta)
	}
	if !hasContent {
		zettel.Content = oldZettel.Content
	}
	return uc.port.UpdateZettel(ctx, zettel)
}
