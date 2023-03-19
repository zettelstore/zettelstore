//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package usecase

import (
	"context"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

// GetAllMetaPort is the interface used by this use case.
type GetAllMetaPort interface {
	// GetAllMeta retrieves just the meta data of a specific zettel.
	GetAllMeta(ctx context.Context, zid id.Zid) ([]*meta.Meta, error)
}

// GetAllMeta is the data for this use case.
type GetAllMeta struct {
	port GetAllMetaPort
}

// NewGetAllMeta creates a new use case.
func NewGetAllMeta(port GetAllMetaPort) GetAllMeta {
	return GetAllMeta{port: port}
}

// Run executes the use case.
func (uc GetAllMeta) Run(ctx context.Context, zid id.Zid) ([]*meta.Meta, error) {
	return uc.port.GetAllMeta(ctx, zid)
}
