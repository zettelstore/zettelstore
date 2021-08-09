//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
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

	"zettelstore.de/z/ast"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/evaluate"
	"zettelstore.de/z/parser"
)

// EvaluateZettel is the data for this use case.
type EvaluateZettel struct {
	rtConfig  config.Config
	getZettel GetZettel
	getMeta   GetMeta
}

// NewEvaluateZettel creates a new use case.
func NewEvaluateZettel(rtConfig config.Config, getZettel GetZettel, getMeta GetMeta) EvaluateZettel {
	return EvaluateZettel{
		rtConfig:  rtConfig,
		getZettel: getZettel,
		getMeta:   getMeta,
	}
}

// Run executes the use case.
func (uc *EvaluateZettel) Run(ctx context.Context, zid id.Zid, env *evaluate.Environment) (*ast.ZettelNode, error) {
	zettel, err := uc.getZettel.Run(ctx, zid)
	if err != nil {
		return nil, err
	}
	zn, err := parser.ParseZettel(zettel, env.Syntax, uc.rtConfig), nil
	if err != nil {
		return nil, err
	}

	evaluate.Evaluate(ctx, uc, env, uc.rtConfig, zn)
	return zn, nil
}

// GetMeta retrieves the metadata of a given zettel identifier.
func (uc *EvaluateZettel) GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	return uc.getMeta.Run(ctx, zid)
}

// GetZettel retrieves the full zettel of a given zettel identifier.
func (uc *EvaluateZettel) GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error) {
	return uc.getZettel.Run(ctx, zid)
}
