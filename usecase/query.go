//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
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
	"errors"

	"zettelstore.de/z/box"
	"zettelstore.de/z/query"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

// QueryPort is the interface used by this use case.
type QueryPort interface {
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
	SelectMeta(ctx context.Context, metaSeq []*meta.Meta, q *query.Query) ([]*meta.Meta, error)
}

// Query is the data for this use case.
type Query struct {
	port QueryPort
}

// NewQuery creates a new use case.
func NewQuery(port QueryPort) Query {
	return Query{port: port}
}

// Run executes the use case.
func (uc *Query) Run(ctx context.Context, q *query.Query) ([]*meta.Meta, error) {
	zids := q.GetZids()
	if zids == nil {
		return uc.port.SelectMeta(ctx, nil, q)
	}
	if len(zids) == 0 {
		return nil, nil
	}
	metaSeq, err := uc.getMetaZid(ctx, zids)
	if err != nil {
		return metaSeq, err
	}
	metaSeq, err = uc.processDirectives(ctx, metaSeq, q.GetDirectives())
	if err != nil {
		return metaSeq, err
	}
	if len(metaSeq) == 0 {
		return nil, nil
	}
	return metaSeq, nil
}

func (uc *Query) getMetaZid(ctx context.Context, zids []id.Zid) ([]*meta.Meta, error) {
	metaSeq := make([]*meta.Meta, 0, len(zids))
	for _, zid := range zids {
		m, err := uc.port.GetMeta(ctx, zid)
		if err == nil {
			metaSeq = append(metaSeq, m)
			continue
		}
		if errors.Is(err, &box.ErrNotAllowed{}) {
			continue
		}
		return metaSeq, err
	}
	return metaSeq, nil
}

func (uc *Query) processDirectives(ctx context.Context, metaSeq []*meta.Meta, directives []query.Directive) ([]*meta.Meta, error) {
	if len(metaSeq) == 0 {
		return nil, nil
	}
	if len(directives) == 0 {
		return metaSeq, nil
	}
	for _, dir := range directives {
		switch ds := dir.(type) {
		case *query.IdentSpec:
			// Nothing to do.
		case *query.ContextSpec:
			metaSeq = ds.Execute(ctx, metaSeq, uc.port)
		default:
			continue
		}
	}
	return metaSeq, nil
}
