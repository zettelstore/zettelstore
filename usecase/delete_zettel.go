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

	"zettelstore.de/z/domain/id"
)

// DeleteZettelPort is the interface used by this use case.
type DeleteZettelPort interface {
	// DeleteZettel removes the zettel from the place.
	DeleteZettel(ctx context.Context, zid id.Zid) error
}

// DeleteZettel is the data for this use case.
type DeleteZettel struct {
	port DeleteZettelPort
}

// NewDeleteZettel creates a new use case.
func NewDeleteZettel(port DeleteZettelPort) DeleteZettel {
	return DeleteZettel{port: port}
}

// Run executes the use case.
func (uc DeleteZettel) Run(ctx context.Context, zid id.Zid) error {
	return uc.port.DeleteZettel(ctx, zid)
}
