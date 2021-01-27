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

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/index"
)

// RenameZettelPort is the interface used by this use case.
type RenameZettelPort interface {
	// GetMeta retrieves just the meta data of a specific zettel.
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)

	// Rename changes the current id to a new id.
	RenameZettel(ctx context.Context, curZid, newZid id.Zid) error
}

// RenameZettel is the data for this use case.
type RenameZettel struct {
	port RenameZettelPort
}

// ErrZidInUse is returned if the zettel id is not appropriate for the place operation.
type ErrZidInUse struct{ Zid id.Zid }

func (err *ErrZidInUse) Error() string {
	return "Zettel id already in use: " + err.Zid.String()
}

// NewRenameZettel creates a new use case.
func NewRenameZettel(port RenameZettelPort) RenameZettel {
	return RenameZettel{port: port}
}

// Run executes the use case.
func (uc RenameZettel) Run(ctx context.Context, curZid, newZid id.Zid) error {
	noEnrichCtx := index.NoEnrichContext(ctx)
	if _, err := uc.port.GetMeta(noEnrichCtx, curZid); err != nil {
		return err
	}
	if _, err := uc.port.GetMeta(noEnrichCtx, newZid); err == nil {
		return &ErrZidInUse{Zid: newZid}
	}
	return uc.port.RenameZettel(ctx, curZid, newZid)
}
