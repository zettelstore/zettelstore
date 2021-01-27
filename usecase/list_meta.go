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

	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
)

// ListMetaPort is the interface used by this use case.
type ListMetaPort interface {
	// SelectMeta returns all zettel meta data that match the selection
	// criteria. The result is ordered by descending zettel id.
	SelectMeta(ctx context.Context, f *place.Filter, s *place.Sorter) ([]*meta.Meta, error)
}

// ListMeta is the data for this use case.
type ListMeta struct {
	port ListMetaPort
}

// NewListMeta creates a new use case.
func NewListMeta(port ListMetaPort) ListMeta {
	return ListMeta{port: port}
}

// Run executes the use case.
func (uc ListMeta) Run(ctx context.Context, f *place.Filter, s *place.Sorter) ([]*meta.Meta, error) {
	return uc.port.SelectMeta(ctx, f, s)
}
