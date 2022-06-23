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
func (uc ListRole) Run(ctx context.Context) (meta.CountedCategories, error) {
	var s *search.Search
	s = s.AddExpr(api.KeyRole, "") // We look for all metadata with a role key
	metas, err := uc.port.SelectMeta(box.NoEnrichContext(ctx), s)
	if err != nil {
		return nil, err
	}
	roleArrangement := make(meta.Arrangement, 256)
	for _, m := range metas {
		role, ok := m.Get(api.KeyRole)
		if !ok {
			panic(m)
		}
		if role == "" {
			panic(m)
		}
		roleArrangement[role] = append(roleArrangement[role], m)
	}
	return roleArrangement.Counted(), nil
}
