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

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/collect"
	"zettelstore.de/z/query"
	"zettelstore.de/z/zettel"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

// QueryPort is the interface used by this use case.
type QueryPort interface {
	GetZettel(ctx context.Context, zid id.Zid) (zettel.Zettel, error)
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
	SelectMeta(ctx context.Context, metaSeq []*meta.Meta, q *query.Query) ([]*meta.Meta, error)
}

// Query is the data for this use case.
type Query struct {
	port       QueryPort
	ucEvaluate Evaluate
}

// NewQuery creates a new use case.
func NewQuery(port QueryPort) Query {
	return Query{port: port}
}

// SetEvaluate sets the usecase Evaluate, because of circular dependencies.
func (uc *Query) SetEvaluate(ucEvaluate *Evaluate) { uc.ucEvaluate = *ucEvaluate }

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
	if metaSeq = uc.processDirectives(ctx, metaSeq, q.GetDirectives()); len(metaSeq) > 0 {
		return uc.port.SelectMeta(ctx, metaSeq, q)
	}
	return nil, nil
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

func (uc *Query) processDirectives(ctx context.Context, metaSeq []*meta.Meta, directives []query.Directive) []*meta.Meta {
	if len(directives) == 0 {
		return metaSeq
	}
	for _, dir := range directives {
		if len(metaSeq) == 0 {
			return nil
		}
		switch ds := dir.(type) {
		case *query.IdentSpec:
			// Nothing to do.
		case *query.ContextSpec:
			metaSeq = ds.Execute(ctx, metaSeq, uc.port)
		case *query.ItemsSpec:
			metaSeq = uc.processItemsDirective(ctx, metaSeq)
		default:
			continue
		}
	}
	if len(metaSeq) == 0 {
		return nil
	}
	return metaSeq
}

func (uc *Query) processItemsDirective(ctx context.Context, metaSeq []*meta.Meta) []*meta.Meta {
	result := make([]*meta.Meta, 0, len(metaSeq))
	for _, m := range metaSeq {
		zn, err := uc.ucEvaluate.Run(ctx, m.Zid, m.GetDefault(api.KeySyntax, ""))
		if err != nil {
			continue
		}
		for _, ref := range collect.Order(zn) {
			if collectedZid, err2 := id.Parse(ref.URL.Path); err2 == nil {
				if z, err3 := uc.port.GetZettel(ctx, collectedZid); err3 == nil {
					result = append(result, z.Meta)
				}
			}
		}
	}
	return result
}
