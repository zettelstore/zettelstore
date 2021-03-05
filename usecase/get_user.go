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

	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/index"
	"zettelstore.de/z/place"
)

// Use case: return user identified by meta key ident.
// ---------------------------------------------------

// GetUserPort is the interface used by this use case.
type GetUserPort interface {
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
	SelectMeta(ctx context.Context, f *place.Filter, s *place.Sorter) ([]*meta.Meta, error)
}

// GetUser is the data for this use case.
type GetUser struct {
	port GetUserPort
}

// NewGetUser creates a new use case.
func NewGetUser(port GetUserPort) GetUser {
	return GetUser{port: port}
}

// Run executes the use case.
func (uc GetUser) Run(ctx context.Context, ident string) (*meta.Meta, error) {
	if !startup.WithAuth() {
		return nil, nil
	}
	ctx = index.NoEnrichContext(ctx)

	// It is important to try first with the owner. First, because another user
	// could give herself the same ''ident''. Second, in most cases the owner
	// will authenticate.
	identMeta, err := uc.port.GetMeta(ctx, startup.Owner())
	if err == nil && identMeta.GetDefault(meta.KeyUserID, "") == ident {
		if role, ok := identMeta.Get(meta.KeyRole); !ok ||
			role != meta.ValueRoleUser {
			return nil, nil
		}
		return identMeta, nil
	}
	// Owner was not found or has another ident. Try via list search.
	filter := place.Filter{
		Expr: map[string][]place.FilterValue{
			meta.KeyRole:   {{Value: meta.ValueRoleUser, Negate: false}},
			meta.KeyUserID: {{Value: ident, Negate: false}},
		},
	}
	metaList, err := uc.port.SelectMeta(ctx, &filter, nil)
	if err != nil {
		return nil, err
	}
	if len(metaList) < 1 {
		return nil, nil
	}
	return metaList[len(metaList)-1], nil
}

// Use case: return an user identified by zettel id and assert given ident value.
// ------------------------------------------------------------------------------

// GetUserByZidPort is the interface used by this use case.
type GetUserByZidPort interface {
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
}

// GetUserByZid is the data for this use case.
type GetUserByZid struct {
	port GetUserByZidPort
}

// NewGetUserByZid creates a new use case.
func NewGetUserByZid(port GetUserByZidPort) GetUserByZid {
	return GetUserByZid{port: port}
}

// Run executes the use case.
func (uc GetUserByZid) Run(ctx context.Context, zid id.Zid, ident string) (*meta.Meta, error) {
	userMeta, err := uc.port.GetMeta(index.NoEnrichContext(ctx), zid)
	if err != nil {
		return nil, err
	}

	if val, ok := userMeta.Get(meta.KeyUserID); !ok || val != ident {
		return nil, nil
	}
	return userMeta, nil
}
