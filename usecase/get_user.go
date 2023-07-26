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

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/z/auth"
	"zettelstore.de/z/box"
	"zettelstore.de/z/query"
	"zettelstore.de/z/zettel"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

// Use case: return user identified by meta key ident.
// ---------------------------------------------------

// GetUserPort is the interface used by this use case.
type GetUserPort interface {
	GetZettel(ctx context.Context, zid id.Zid) (zettel.Zettel, error)
	SelectMeta(ctx context.Context, q *query.Query) ([]*meta.Meta, error)
}

// GetUser is the data for this use case.
type GetUser struct {
	authz auth.AuthzManager
	port  GetUserPort
}

// NewGetUser creates a new use case.
func NewGetUser(authz auth.AuthzManager, port GetUserPort) GetUser {
	return GetUser{authz: authz, port: port}
}

// Run executes the use case.
func (uc GetUser) Run(ctx context.Context, ident string) (*meta.Meta, error) {
	ctx = box.NoEnrichContext(ctx)

	// It is important to try first with the owner. First, because another user
	// could give herself the same ''ident''. Second, in most cases the owner
	// will authenticate.
	identZettel, err := uc.port.GetZettel(ctx, uc.authz.Owner())
	if err == nil && identZettel.Meta.GetDefault(api.KeyUserID, "") == ident {
		return identZettel.Meta, nil
	}
	// Owner was not found or has another ident. Try via list search.
	q := query.Parse(api.KeyUserID + api.SearchOperatorHas + ident + " " + api.SearchOperatorHas + ident)
	metaList, err := uc.port.SelectMeta(ctx, q)
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
	GetZettel(ctx context.Context, zid id.Zid) (zettel.Zettel, error)
}

// GetUserByZid is the data for this use case.
type GetUserByZid struct {
	port GetUserByZidPort
}

// NewGetUserByZid creates a new use case.
func NewGetUserByZid(port GetUserByZidPort) GetUserByZid {
	return GetUserByZid{port: port}
}

// GetUser executes the use case.
func (uc GetUserByZid) GetUser(ctx context.Context, zid id.Zid, ident string) (*meta.Meta, error) {
	userZettel, err := uc.port.GetZettel(box.NoEnrichContext(ctx), zid)
	if err != nil {
		return nil, err
	}

	userMeta := userZettel.Meta
	if val, ok := userMeta.Get(api.KeyUserID); !ok || val != ident {
		return nil, nil
	}
	return userMeta, nil
}
