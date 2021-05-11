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
	"sort"

	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/search"
)

// ListRolePort is the interface used by this use case.
type ListRolePort interface {
	// SelectMeta returns all zettel meta data that match the selection criteria.
	SelectMeta(ctx context.Context, s *search.Search) ([]*meta.Meta, error)
}

// ListRole is the data for this use case.
type ListRole struct {
	port ListRolePort
}

// NewListRole creates a new use case.
func NewListRole(port ListRolePort) ListRole {
	return ListRole{port: port}
}

// Run executes the use case.
func (uc ListRole) Run(ctx context.Context) ([]string, error) {
	metas, err := uc.port.SelectMeta(place.NoEnrichContext(ctx), nil)
	if err != nil {
		return nil, err
	}
	roles := make(map[string]bool, 8)
	for _, m := range metas {
		if role, ok := m.Get(meta.KeyRole); ok && role != "" {
			roles[role] = true
		}
	}
	result := make([]string, 0, len(roles))
	for role := range roles {
		result = append(result, role)
	}
	sort.Strings(result)
	return result, nil
}
