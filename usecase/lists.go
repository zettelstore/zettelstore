//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
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

	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/search"
)

// ListMetaPort is the interface used by this use case.
type ListMetaPort interface {
	// SelectMeta returns all zettel metadata that match the selection criteria.
	SelectMeta(ctx context.Context, s *search.Search) ([]*meta.Meta, error)
}

// ListMeta is the data for this use case.
type ListMeta struct {
	port ListMetaPort
}

// NewListMeta creates a new use case.
func NewListMeta(port ListMetaPort) ListMeta {
	return ListMeta{port: port}
}

// Run executes the use case.
func (uc ListMeta) Run(ctx context.Context, s *search.Search) ([]*meta.Meta, error) {
	return uc.port.SelectMeta(ctx, s)
}

// -------- List roles -------------------------------------------------------

// ListRolesPort is the interface used by this use case.
type ListRolesPort interface {
	// SelectMeta returns all zettel metadata that match the selection criteria.
	SelectMeta(ctx context.Context, s *search.Search) ([]*meta.Meta, error)
}

// ListRoles is the data for this use case.
type ListRoles struct {
	port ListRolesPort
}

// NewListRoles creates a new use case.
func NewListRoles(port ListRolesPort) ListRoles {
	return ListRoles{port: port}
}

// Run executes the use case.
func (uc ListRoles) Run(ctx context.Context) (meta.Arrangement, error) {
	var s *search.Search
	s = s.AddExpr(api.KeyRole, "") // We look for all metadata with a role key
	metas, err := uc.port.SelectMeta(box.NoEnrichContext(ctx), s)
	if err != nil {
		return nil, err
	}
	return meta.CreateArrangement(metas, api.KeyRole), nil
}

// -------- List tags --------------------------------------------------------

// ListTagsPort is the interface used by this use case.
type ListTagsPort interface {
	// SelectMeta returns all zettel metadata that match the selection criteria.
	SelectMeta(ctx context.Context, s *search.Search) ([]*meta.Meta, error)
}

// ListTags is the data for this use case.
type ListTags struct {
	port ListTagsPort
}

// NewListTags creates a new use case.
func NewListTags(port ListTagsPort) ListTags {
	return ListTags{port: port}
}

// Run executes the use case.
func (uc ListTags) Run(ctx context.Context, minCount int) (meta.Arrangement, error) {
	var s *search.Search
	s = s.AddExpr(api.KeyTags, "") // We look for all metadata with a tag
	metas, err := uc.port.SelectMeta(ctx, s)
	if err != nil {
		return nil, err
	}
	result := meta.CreateArrangement(metas, api.KeyAllTags)
	if minCount > 1 {
		for t, ms := range result {
			if len(ms) < minCount {
				delete(result, t)
			}
		}
	}
	return result, nil
}
