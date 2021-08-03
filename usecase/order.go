//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package usecase provides (business) use cases for the Zettelstore.
package usecase

import (
	"context"

	"zettelstore.de/z/collect"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/evaluate"
)

// ZettelOrderPort is the interface used by this use case.
type ZettelOrderPort interface {
	// GetMeta retrieves just the meta data of a specific zettel.
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
}

// ZettelOrder is the data for this use case.
type ZettelOrder struct {
	port           ZettelOrderPort
	evaluateZettel EvaluateZettel
}

// NewZettelOrder creates a new use case.
func NewZettelOrder(port ZettelOrderPort, evaluateZettel EvaluateZettel) ZettelOrder {
	return ZettelOrder{port: port, evaluateZettel: evaluateZettel}
}

// Run executes the use case.
func (uc ZettelOrder) Run(ctx context.Context, zid id.Zid, syntax string) (
	start *meta.Meta, result []*meta.Meta, err error,
) {
	zn, err := uc.evaluateZettel.Run(ctx, zid, &evaluate.Environment{
		Syntax: syntax,
	})
	if err != nil {
		return nil, nil, err
	}
	for _, ref := range collect.Order(zn) {
		if zid, err := id.Parse(ref.URL.Path); err == nil {
			if m, err := uc.port.GetMeta(ctx, zid); err == nil {
				result = append(result, m)
			}
		}
	}
	return zn.Meta, result, nil
}
