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
	"zettelstore.de/z/domain/meta"
)

// GetMetaPort is the interface used by this use case.
type GetMetaPort interface {
	// GetMeta retrieves just the meta data of a specific zettel.
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
}

// GetMeta is the data for this use case.
type GetMeta struct {
	port GetMetaPort
}

// NewGetMeta creates a new use case.
func NewGetMeta(port GetMetaPort) GetMeta {
	return GetMeta{port: port}
}

// Run executes the use case.
func (uc GetMeta) Run(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	return uc.port.GetMeta(ctx, zid)
}
