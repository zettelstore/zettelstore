//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package policy

import (
	"context"

	"zettelstore.de/z/auth"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/search"
	"zettelstore.de/z/web/server"
)

// BoxWithPolicy wraps the given box inside a policy box.
func BoxWithPolicy(
	auth server.Auth,
	manager auth.AuthzManager,
	box box.Box,
	authConfig config.AuthConfig,
) (box.Box, auth.Policy) {
	pol := newPolicy(manager, authConfig)
	return newBox(auth, box, pol), pol
}

// polBox implements a policy box.
type polBox struct {
	auth   server.Auth
	box    box.Box
	policy auth.Policy
}

// newBox creates a new policy box.
func newBox(auth server.Auth, box box.Box, policy auth.Policy) box.Box {
	return &polBox{
		auth:   auth,
		box:    box,
		policy: policy,
	}
}

func (pp *polBox) Location() string {
	return pp.box.Location()
}

func (pp *polBox) CanCreateZettel(ctx context.Context) bool {
	return pp.box.CanCreateZettel(ctx)
}

func (pp *polBox) CreateZettel(ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	user := pp.auth.GetUser(ctx)
	if pp.policy.CanCreate(user, zettel.Meta) {
		return pp.box.CreateZettel(ctx, zettel)
	}
	return id.Invalid, box.NewErrNotAllowed("Create", user, id.Invalid)
}

func (pp *polBox) GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error) {
	zettel, err := pp.box.GetZettel(ctx, zid)
	if err != nil {
		return domain.Zettel{}, err
	}
	user := pp.auth.GetUser(ctx)
	if pp.policy.CanRead(user, zettel.Meta) {
		return zettel, nil
	}
	return domain.Zettel{}, box.NewErrNotAllowed("GetZettel", user, zid)
}

func (pp *polBox) GetAllZettel(ctx context.Context, zid id.Zid) ([]domain.Zettel, error) {
	return pp.box.GetAllZettel(ctx, zid)
}

func (pp *polBox) GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	m, err := pp.box.GetMeta(ctx, zid)
	if err != nil {
		return nil, err
	}
	user := pp.auth.GetUser(ctx)
	if pp.policy.CanRead(user, m) {
		return m, nil
	}
	return nil, box.NewErrNotAllowed("GetMeta", user, zid)
}

func (pp *polBox) GetAllMeta(ctx context.Context, zid id.Zid) ([]*meta.Meta, error) {
	return pp.box.GetAllMeta(ctx, zid)
}

func (pp *polBox) FetchZids(ctx context.Context) (id.Set, error) {
	return nil, box.NewErrNotAllowed("fetch-zids", pp.auth.GetUser(ctx), id.Invalid)
}

func (pp *polBox) SelectMeta(ctx context.Context, s *search.Search) ([]*meta.Meta, error) {
	user := pp.auth.GetUser(ctx)
	canRead := pp.policy.CanRead
	s = s.SetPreMatch(func(m *meta.Meta) bool { return canRead(user, m) })
	return pp.box.SelectMeta(ctx, s)
}

func (pp *polBox) CanUpdateZettel(ctx context.Context, zettel domain.Zettel) bool {
	return pp.box.CanUpdateZettel(ctx, zettel)
}

func (pp *polBox) UpdateZettel(ctx context.Context, zettel domain.Zettel) error {
	zid := zettel.Meta.Zid
	user := pp.auth.GetUser(ctx)
	if !zid.IsValid() {
		return &box.ErrInvalidID{Zid: zid}
	}
	// Write existing zettel
	oldMeta, err := pp.box.GetMeta(ctx, zid)
	if err != nil {
		return err
	}
	if pp.policy.CanWrite(user, oldMeta, zettel.Meta) {
		return pp.box.UpdateZettel(ctx, zettel)
	}
	return box.NewErrNotAllowed("Write", user, zid)
}

func (pp *polBox) AllowRenameZettel(ctx context.Context, zid id.Zid) bool {
	return pp.box.AllowRenameZettel(ctx, zid)
}

func (pp *polBox) RenameZettel(ctx context.Context, curZid, newZid id.Zid) error {
	meta, err := pp.box.GetMeta(ctx, curZid)
	if err != nil {
		return err
	}
	user := pp.auth.GetUser(ctx)
	if pp.policy.CanRename(user, meta) {
		return pp.box.RenameZettel(ctx, curZid, newZid)
	}
	return box.NewErrNotAllowed("Rename", user, curZid)
}

func (pp *polBox) CanDeleteZettel(ctx context.Context, zid id.Zid) bool {
	return pp.box.CanDeleteZettel(ctx, zid)
}

func (pp *polBox) DeleteZettel(ctx context.Context, zid id.Zid) error {
	meta, err := pp.box.GetMeta(ctx, zid)
	if err != nil {
		return err
	}
	user := pp.auth.GetUser(ctx)
	if pp.policy.CanDelete(user, meta) {
		return pp.box.DeleteZettel(ctx, zid)
	}
	return box.NewErrNotAllowed("Delete", user, zid)
}

func (pp *polBox) Refresh(ctx context.Context) error {
	user := pp.auth.GetUser(ctx)
	if pp.policy.CanRefresh(user) {
		return pp.box.Refresh(ctx)
	}
	return box.NewErrNotAllowed("Refresh", user, id.Invalid)
}
