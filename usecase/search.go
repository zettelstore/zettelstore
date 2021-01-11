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

	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
)

// SearchPort is the interface used by this use case.
type SearchPort interface {
	// SelectMeta returns all zettel meta data that match the selection
	// criteria. The result is ordered by descending zettel id.
	SelectMeta(ctx context.Context, f *place.Filter, s *place.Sorter) ([]*meta.Meta, error)
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
func (uc Search) Run(
	ctx context.Context, f *place.Filter, s *place.Sorter) ([]*meta.Meta, error) {
	// TODO: interpret f[""]. Can contain expressions for specific meta tags.
	return uc.port.SelectMeta(ctx, f, s)
}
