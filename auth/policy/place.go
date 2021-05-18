//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package policy provides some interfaces and implementation for authorizsation policies.
package policy

import (
	"context"
	"io"

	"zettelstore.de/z/auth"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/search"
	"zettelstore.de/z/web/server"
)

// PlaceWithPolicy wraps the given place inside a policy place.
func PlaceWithPolicy(
	auth server.Auth,
	manager auth.AuthzManager,
	place place.Place,
	expertMode func() bool,
	getVisibility func(*meta.Meta) meta.Visibility,
) (place.Place, auth.Policy) {
	pol := newPolicy(manager, expertMode, getVisibility)
	return newPlace(auth, place, pol), pol
}

// polPlace implements a policy place.
type polPlace struct {
	auth   server.Auth
	place  place.Place
	policy auth.Policy
}

// newPlace creates a new policy place.
func newPlace(auth server.Auth, place place.Place, policy auth.Policy) place.Place {
	return &polPlace{
		auth:   auth,
		place:  place,
		policy: policy,
	}
}

func (pp *polPlace) Location() string {
	return pp.place.Location()
}

func (pp *polPlace) CanCreateZettel(ctx context.Context) bool {
	return pp.place.CanCreateZettel(ctx)
}

func (pp *polPlace) CreateZettel(ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	user := pp.auth.GetUser(ctx)
	if pp.policy.CanCreate(user, zettel.Meta) {
		return pp.place.CreateZettel(ctx, zettel)
	}
	return id.Invalid, place.NewErrNotAllowed("Create", user, id.Invalid)
}

func (pp *polPlace) GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error) {
	zettel, err := pp.place.GetZettel(ctx, zid)
	if err != nil {
		return domain.Zettel{}, err
	}
	user := pp.auth.GetUser(ctx)
	if pp.policy.CanRead(user, zettel.Meta) {
		return zettel, nil
	}
	return domain.Zettel{}, place.NewErrNotAllowed("GetZettel", user, zid)
}

func (pp *polPlace) GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	m, err := pp.place.GetMeta(ctx, zid)
	if err != nil {
		return nil, err
	}
	user := pp.auth.GetUser(ctx)
	if pp.policy.CanRead(user, m) {
		return m, nil
	}
	return nil, place.NewErrNotAllowed("GetMeta", user, zid)
}

func (pp *polPlace) FetchZids(ctx context.Context) (id.Set, error) {
	return nil, place.NewErrNotAllowed("fetch-zids", pp.auth.GetUser(ctx), id.Invalid)
}

func (pp *polPlace) SelectMeta(ctx context.Context, s *search.Search) ([]*meta.Meta, error) {
	user := pp.auth.GetUser(ctx)
	canRead := pp.policy.CanRead
	s = s.AddPreMatch(func(m *meta.Meta) bool { return canRead(user, m) })
	return pp.place.SelectMeta(ctx, s)
}

func (pp *polPlace) CanUpdateZettel(ctx context.Context, zettel domain.Zettel) bool {
	return pp.place.CanUpdateZettel(ctx, zettel)
}

func (pp *polPlace) UpdateZettel(ctx context.Context, zettel domain.Zettel) error {
	zid := zettel.Meta.Zid
	user := pp.auth.GetUser(ctx)
	if !zid.IsValid() {
		return &place.ErrInvalidID{Zid: zid}
	}
	// Write existing zettel
	oldMeta, err := pp.place.GetMeta(ctx, zid)
	if err != nil {
		return err
	}
	if pp.policy.CanWrite(user, oldMeta, zettel.Meta) {
		return pp.place.UpdateZettel(ctx, zettel)
	}
	return place.NewErrNotAllowed("Write", user, zid)
}

func (pp *polPlace) AllowRenameZettel(ctx context.Context, zid id.Zid) bool {
	return pp.place.AllowRenameZettel(ctx, zid)
}

func (pp *polPlace) RenameZettel(ctx context.Context, curZid, newZid id.Zid) error {
	meta, err := pp.place.GetMeta(ctx, curZid)
	if err != nil {
		return err
	}
	user := pp.auth.GetUser(ctx)
	if pp.policy.CanRename(user, meta) {
		return pp.place.RenameZettel(ctx, curZid, newZid)
	}
	return place.NewErrNotAllowed("Rename", user, curZid)
}

func (pp *polPlace) CanDeleteZettel(ctx context.Context, zid id.Zid) bool {
	return pp.place.CanDeleteZettel(ctx, zid)
}

func (pp *polPlace) DeleteZettel(ctx context.Context, zid id.Zid) error {
	meta, err := pp.place.GetMeta(ctx, zid)
	if err != nil {
		return err
	}
	user := pp.auth.GetUser(ctx)
	if pp.policy.CanDelete(user, meta) {
		return pp.place.DeleteZettel(ctx, zid)
	}
	return place.NewErrNotAllowed("Delete", user, zid)
}

func (pp *polPlace) ReadStats(st *place.Stats) {
	pp.place.ReadStats(st)
}

func (pp *polPlace) Dump(w io.Writer) {
	pp.place.Dump(w)
}
