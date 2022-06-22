//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
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

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/search"
)

// ListTagsPort is the interface used by this use case.
type ListTagsPort interface {
	// SelectMeta returns all zettel meta data that match the selection criteria.
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
	metas, err := uc.port.SelectMeta(ctx, nil)
	if err != nil {
		return nil, err
	}
	result := make(meta.Arrangement)
	for _, m := range metas {
		if tl, ok := m.GetList(api.KeyAllTags); ok && len(tl) > 0 {
			for _, t := range tl {
				result[t] = append(result[t], m)
			}
		}
	}
	if minCount > 1 {
		for t, ms := range result {
			if len(ms) < minCount {
				delete(result, t)
			}
		}
	}
	return result, nil
}
