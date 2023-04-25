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

	"zettelstore.de/z/collect"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

// ZettelOrderPort is the interface used by this use case.
type ZettelOrderPort interface {
	// GetMeta retrieves just the meta data of a specific zettel.
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
}

// ZettelOrder is the data for this use case.
type ZettelOrder struct {
	port     ZettelOrderPort
	evaluate Evaluate
}

// NewZettelOrder creates a new use case.
func NewZettelOrder(port ZettelOrderPort, evaluate Evaluate) ZettelOrder {
	return ZettelOrder{port: port, evaluate: evaluate}
}

// Run executes the use case.
func (uc ZettelOrder) Run(ctx context.Context, zid id.Zid, syntax string) (
	start *meta.Meta, result []*meta.Meta, err error,
) {
	zn, err := uc.evaluate.Run(ctx, zid, syntax)
	if err != nil {
		return nil, nil, err
	}
	for _, ref := range collect.Order(zn) {
		if collectedZid, err2 := id.Parse(ref.URL.Path); err2 == nil {
			if m, err3 := uc.port.GetMeta(ctx, collectedZid); err3 == nil {
				result = append(result, m)
			}
		}
	}
	return zn.Meta, result, nil
}
