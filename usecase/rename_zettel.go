//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2020-present Detlef Stern
//-----------------------------------------------------------------------------

package usecase

import (
	"context"

	"zettelstore.de/z/box"
	"zettelstore.de/z/logger"
	"zettelstore.de/z/zettel"
	"zettelstore.de/z/zettel/id"
)

// RenameZettelPort is the interface used by this use case.
type RenameZettelPort interface {
	GetZettel(ctx context.Context, zid id.ZidO) (zettel.Zettel, error)
	RenameZettel(ctx context.Context, curZid, newZid id.ZidO) error
}

// RenameZettel is the data for this use case.
type RenameZettel struct {
	log  *logger.Logger
	port RenameZettelPort
}

// ErrZidInUse is returned if the zettel id is not appropriate for the box operation.
type ErrZidInUse struct{ Zid id.ZidO }

func (err ErrZidInUse) Error() string {
	return "Zettel id already in use: " + err.Zid.String()
}

// NewRenameZettel creates a new use case.
func NewRenameZettel(log *logger.Logger, port RenameZettelPort) RenameZettel {
	return RenameZettel{log: log, port: port}
}

// Run executes the use case.
func (uc *RenameZettel) Run(ctx context.Context, curZid, newZid id.ZidO) error {
	noEnrichCtx := box.NoEnrichContext(ctx)
	if _, err := uc.port.GetZettel(noEnrichCtx, curZid); err != nil {
		return err
	}
	if newZid == curZid {
		// Nothing to do
		return nil
	}
	if _, err := uc.port.GetZettel(noEnrichCtx, newZid); err == nil {
		return ErrZidInUse{Zid: newZid}
	}
	err := uc.port.RenameZettel(ctx, curZid, newZid)
	uc.log.Info().User(ctx).Zid(curZid).Err(err).Zid(newZid).Msg("Rename zettel")
	return err
}
