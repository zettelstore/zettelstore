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

	"zettelstore.de/z/ast"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/evaluator"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/query"
)

// Evaluate is the data for this use case.
type Evaluate struct {
	rtConfig  config.Config
	getZettel GetZettel
	getMeta   GetMeta
	listMeta  ListMeta
}

// NewEvaluate creates a new use case.
func NewEvaluate(rtConfig config.Config, getZettel GetZettel, getMeta GetMeta, listMeta ListMeta) Evaluate {
	return Evaluate{
		rtConfig:  rtConfig,
		getZettel: getZettel,
		getMeta:   getMeta,
		listMeta:  listMeta,
	}
}

// Run executes the use case.
func (uc *Evaluate) Run(ctx context.Context, zid id.Zid, syntax string) (*ast.ZettelNode, error) {
	zettel, err := uc.getZettel.Run(ctx, zid)
	if err != nil {
		return nil, err
	}
	zn, err := parser.ParseZettel(ctx, zettel, syntax, uc.rtConfig), nil
	if err != nil {
		return nil, err
	}

	evaluator.EvaluateZettel(ctx, uc, uc.rtConfig, zn)
	return zn, nil
}

// RunBlockNode executes the use case for a metadata list.
func (uc *Evaluate) RunBlockNode(ctx context.Context, bn ast.BlockNode) ast.BlockSlice {
	if bn == nil {
		return nil
	}
	bns := ast.BlockSlice{bn}
	evaluator.EvaluateBlock(ctx, uc, uc.rtConfig, &bns)
	return bns
}

// RunMetadata executes the use case for a metadata value.
func (uc *Evaluate) RunMetadata(ctx context.Context, value string) ast.InlineSlice {
	is := parser.ParseMetadata(value)
	evaluator.EvaluateInline(ctx, uc, uc.rtConfig, &is)
	return is
}

// GetMeta retrieves the metadata of a given zettel identifier.
func (uc *Evaluate) GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	return uc.getMeta.Run(ctx, zid)
}

// GetZettel retrieves the full zettel of a given zettel identifier.
func (uc *Evaluate) GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error) {
	return uc.getZettel.Run(ctx, zid)
}

// SelectMeta returns a list of metadata that comply to the given selection criteria.
func (uc *Evaluate) SelectMeta(ctx context.Context, q *query.Query) ([]*meta.Meta, error) {
	return uc.listMeta.Run(ctx, q)
}
