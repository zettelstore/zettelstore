//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package usecase

import (
	"context"

	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/logger"
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
	log  *logger.Logger
	port UpdateZettelPort
}

// NewUpdateZettel creates a new use case.
func NewUpdateZettel(log *logger.Logger, port UpdateZettelPort) UpdateZettel {
	return UpdateZettel{log: log, port: port}
}

// Run executes the use case.
func (uc *UpdateZettel) Run(ctx context.Context, zettel domain.Zettel, hasContent bool) error {
	m := zettel.Meta
	oldZettel, err := uc.port.GetZettel(box.NoEnrichContext(ctx), m.Zid)
	if err != nil {
		return err
	}
	if zettel.Equal(oldZettel, false) {
		return nil
	}
	m.SetNow(api.KeyModified)
	m.YamlSep = oldZettel.Meta.YamlSep
	if m.Zid == id.ConfigurationZid {
		m.Set(api.KeySyntax, api.ValueSyntaxNone)
	}
	if !hasContent {
		zettel.Content = oldZettel.Content
		zettel.Content.TrimSpace()
	}
	err = uc.port.UpdateZettel(ctx, zettel)
	uc.log.Debug().User(ctx).Zid(m.Zid).Err(err).Msg("Update zettel")
	return err
}
