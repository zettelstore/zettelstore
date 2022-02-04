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
	"sort"

	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/search"
	"zettelstore.de/z/strfun"
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
	metas, err := uc.port.SelectMeta(box.NoEnrichContext(ctx), nil)
	if err != nil {
		return nil, err
	}
	roles := make(strfun.Set, 256)
	for _, m := range metas {
		if role, ok := m.Get(api.KeyRole); ok && role != "" {
			roles.Set(role)
		}
	}
	result := make([]string, 0, len(roles))
	for role := range roles {
		result = append(result, role)
	}
	sort.Strings(result)
	return result, nil
}
