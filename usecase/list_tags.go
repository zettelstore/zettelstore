//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
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

	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/index"
	"zettelstore.de/z/place"
)

// ListTagsPort is the interface used by this use case.
type ListTagsPort interface {
	// SelectMeta returns all zettel meta data that match the selection
	// criteria. The result is ordered by descending zettel id.
	SelectMeta(ctx context.Context, f *place.Filter, s *place.Sorter) ([]*meta.Meta, error)
}

// ListTags is the data for this use case.
type ListTags struct {
	port ListTagsPort
}

// NewListTags creates a new use case.
func NewListTags(port ListTagsPort) ListTags {
	return ListTags{port: port}
}

// TagData associates tags with a list of all zettel meta that use this tag
type TagData map[string][]*meta.Meta

// Run executes the use case.
func (uc ListTags) Run(ctx context.Context, minCount int) (TagData, error) {
	metas, err := uc.port.SelectMeta(index.NoEnrichContext(ctx), nil, nil)
	if err != nil {
		return nil, err
	}
	result := make(TagData)
	for _, m := range metas {
		if tl, ok := m.GetList(meta.KeyTags); ok && len(tl) > 0 {
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
