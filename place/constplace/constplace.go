//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package constplace places zettel inside the executable.
package constplace

import (
	"context"
	"net/url"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/index"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/manager"
)

func init() {
	manager.Register(
		" const",
		func(u *url.URL, cdata *manager.ConnectData) (place.Place, error) {
			return &constPlace{zettel: constZettelMap, filter: cdata.Filter}, nil
		})
}

type constHeader map[string]string

func makeMeta(zid id.Zid, h constHeader) *meta.Meta {
	m := meta.New(zid)
	for k, v := range h {
		m.Set(k, v)
	}
	return m
}

type constZettel struct {
	header  constHeader
	content domain.Content
}

type constPlace struct {
	zettel map[id.Zid]constZettel
	filter index.MetaFilter
}

func (cp *constPlace) Location() string {
	return "const:"
}

func (cp *constPlace) CanCreateZettel(ctx context.Context) bool { return false }

func (cp *constPlace) CreateZettel(
	ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	return id.Invalid, place.ErrReadOnly
}

func (cp *constPlace) GetZettel(
	ctx context.Context, zid id.Zid) (domain.Zettel, error) {
	if z, ok := cp.zettel[zid]; ok {
		return domain.Zettel{Meta: makeMeta(zid, z.header), Content: z.content}, nil
	}
	return domain.Zettel{}, place.ErrNotFound
}

func (cp *constPlace) GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	if z, ok := cp.zettel[zid]; ok {
		return makeMeta(zid, z.header), nil
	}
	return nil, place.ErrNotFound
}

func (cp *constPlace) FetchZids(ctx context.Context) (map[id.Zid]bool, error) {
	result := make(map[id.Zid]bool, len(cp.zettel))
	for zid := range cp.zettel {
		result[zid] = true
	}
	return result, nil
}

func (cp *constPlace) SelectMeta(
	ctx context.Context, f *place.Filter, s *place.Sorter) (res []*meta.Meta, err error) {
	hasMatch := place.CreateFilterFunc(f)
	for zid, zettel := range cp.zettel {
		m := makeMeta(zid, zettel.header)
		cp.filter.Update(ctx, m)
		if hasMatch(m) {
			res = append(res, m)
		}
	}
	return place.ApplySorter(res, s), nil
}

func (cp *constPlace) CanUpdateZettel(ctx context.Context, zettel domain.Zettel) bool {
	return false
}

func (cp *constPlace) UpdateZettel(ctx context.Context, zettel domain.Zettel) error {
	return place.ErrReadOnly
}

func (cp *constPlace) AllowRenameZettel(ctx context.Context, zid id.Zid) bool {
	_, ok := cp.zettel[zid]
	return !ok
}

func (cp *constPlace) RenameZettel(ctx context.Context, curZid, newZid id.Zid) error {
	if _, ok := cp.zettel[curZid]; ok {
		return place.ErrReadOnly
	}
	return place.ErrNotFound
}
func (cp *constPlace) CanDeleteZettel(ctx context.Context, zid id.Zid) bool { return false }

func (cp *constPlace) DeleteZettel(ctx context.Context, zid id.Zid) error {
	if _, ok := cp.zettel[zid]; ok {
		return place.ErrReadOnly
	}
	return place.ErrNotFound
}

func (cp *constPlace) Reload(ctx context.Context) error { return nil }

func (cp *constPlace) ReadStats(st *place.Stats) {
	st.ReadOnly = true
	st.Zettel = len(cp.zettel)
}
