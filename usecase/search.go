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

	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/search"
)

// SearchPort is the interface used by this use case.
type SearchPort interface {
	// SelectMeta returns all zettel meta data that match the selection criteria.
	SelectMeta(ctx context.Context, s *search.Search) ([]*meta.Meta, error)
}

// Search is the data for this use case.
type Search struct {
	port SearchPort
}

// NewSearch creates a new use case.
func NewSearch(port SearchPort) Search {
	return Search{port: port}
}

// Run executes the use case.
func (uc Search) Run(ctx context.Context, s *search.Search) ([]*meta.Meta, error) {
	if !s.EnrichNeeded() {
		ctx = box.NoEnrichContext(ctx)
	}
	return uc.port.SelectMeta(ctx, s)
}
