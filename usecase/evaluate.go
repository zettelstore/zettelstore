//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package usecase

import (
	"context"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/evaluator"
	"zettelstore.de/z/parser"
)

// Evaluate is the data for this use case.
type Evaluate struct {
	rtConfig  config.Config
	getZettel GetZettel
	getMeta   GetMeta
}

// NewEvaluate creates a new use case.
func NewEvaluate(rtConfig config.Config, getZettel GetZettel, getMeta GetMeta) Evaluate {
	return Evaluate{
		rtConfig:  rtConfig,
		getZettel: getZettel,
		getMeta:   getMeta,
	}
}

// Run executes the use case.
func (uc *Evaluate) Run(ctx context.Context, zid id.Zid, syntax string, env *evaluator.Environment) (*ast.ZettelNode, error) {
	zettel, err := uc.getZettel.Run(ctx, zid)
	if err != nil {
		return nil, err
	}
	zn, err := parser.ParseZettel(zettel, syntax, uc.rtConfig), nil
	if err != nil {
		return nil, err
	}

	evaluator.EvaluateZettel(ctx, uc, env, uc.rtConfig, zn)
	return zn, nil
}

// RunMetadata executes the use case for a metadata value.
func (uc *Evaluate) RunMetadata(ctx context.Context, value string, env *evaluator.Environment) *ast.InlineListNode {
	iln := parser.ParseMetadata(value)
	evaluator.EvaluateInline(ctx, uc, env, uc.rtConfig, iln)
	return iln
}

// GetMeta retrieves the metadata of a given zettel identifier.
func (uc *Evaluate) GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	return uc.getMeta.Run(ctx, zid)
}

// GetZettel retrieves the full zettel of a given zettel identifier.
func (uc *Evaluate) GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error) {
	return uc.getZettel.Run(ctx, zid)
}
