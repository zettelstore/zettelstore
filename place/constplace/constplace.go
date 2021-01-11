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
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/manager"
)

func init() {
	manager.Register(
		" const",
		func(u *url.URL, mf manager.MetaFilter) (place.Place, error) {
			return &constPlace{zettel: constZettelMap, filter: mf}, nil
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
	filter manager.MetaFilter
}

// Location returns some information where the place is located.
func (cp *constPlace) Location() string {
	return "const:"
}

// Start the place. Now all other functions of the place are allowed.
// Starting an already started place is not allowed.
func (cp *constPlace) Start(ctx context.Context) error { return nil }

// Stop the started place. Now only the Start() function is allowed.
func (cp *constPlace) Stop(ctx context.Context) error { return nil }

// RegisterChangeObserver registers an observer that will be notified
// if a zettel was found to be changed.
// This place never changes anything. So ignore the registration.
func (cp *constPlace) RegisterChangeObserver(f place.ObserverFunc) {}

func (cp *constPlace) CanCreateZettel(ctx context.Context) bool { return false }

func (cp *constPlace) CreateZettel(
	ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	return id.Invalid, place.ErrReadOnly
}

// GetZettel retrieves a specific zettel.
func (cp *constPlace) GetZettel(
	ctx context.Context, zid id.Zid) (domain.Zettel, error) {
	if z, ok := cp.zettel[zid]; ok {
		return domain.Zettel{Meta: makeMeta(zid, z.header), Content: z.content}, nil
	}
	return domain.Zettel{}, place.ErrNotFound
}

// GetMeta retrieves just the meta data of a specific zettel.
func (cp *constPlace) GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	if z, ok := cp.zettel[zid]; ok {
		return makeMeta(zid, z.header), nil
	}
	return nil, place.ErrNotFound
}

// SelectMeta returns all zettel meta data that match the selection
// criteria. The result is ordered by descending zettel id.
func (cp *constPlace) SelectMeta(
	ctx context.Context, f *place.Filter, s *place.Sorter) (res []*meta.Meta, err error) {
	hasMatch := place.CreateFilterFunc(f)
	for zid, zettel := range cp.zettel {
		m := makeMeta(zid, zettel.header)
		cp.filter.UpdateProperties(m)
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

// Rename changes the current id to a new id.
func (cp *constPlace) RenameZettel(ctx context.Context, curZid, newZid id.Zid) error {
	if _, ok := cp.zettel[curZid]; ok {
		return place.ErrReadOnly
	}
	return place.ErrNotFound
}
func (cp *constPlace) CanDeleteZettel(ctx context.Context, zid id.Zid) bool { return false }

// DeleteZettel removes the zettel from the place.
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
