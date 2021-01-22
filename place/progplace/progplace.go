//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package progplace provides zettel that inform the user about the internal
// Zettelstore state.
package progplace

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
		" prog",
		func(u *url.URL, cdata *manager.ConnectData) (place.Place, error) {
			return getPlace(cdata.Filter), nil
		})
}

type (
	zettelGen struct {
		meta    func(id.Zid) *meta.Meta
		content func(*meta.Meta) string
	}

	progPlace struct {
		zettel      map[id.Zid]zettelGen
		filter      index.MetaFilter
		startConfig *meta.Meta
		manager     place.Manager
		indexer     index.Indexer
	}
)

var myPlace *progPlace

// Get returns the one program place.
func getPlace(mf index.MetaFilter) place.Place {
	if myPlace == nil {
		myPlace = &progPlace{
			zettel: map[id.Zid]zettelGen{
				id.Zid(1):  {genVersionBuildM, genVersionBuildC},
				id.Zid(2):  {genVersionHostM, genVersionHostC},
				id.Zid(3):  {genVersionOSM, genVersionOSC},
				id.Zid(6):  {genEnvironmentM, genEnvironmentC},
				id.Zid(8):  {genRuntimeM, genRuntimeC},
				id.Zid(20): {genManagerM, genManagerC},
				id.Zid(90): {genKeysM, genKeysC},
				id.Zid(96): {genConfigZettelM, genConfigZettelC},
				id.Zid(98): {genConfigM, genConfigC},
			},
			filter: mf,
		}
	}
	return myPlace
}

// Setup remembers important values.
func Setup(startConfig *meta.Meta, manager place.Manager, idx index.Indexer) {
	if myPlace == nil {
		panic("progplace.getPlace not called")
	}
	if myPlace.startConfig != nil || myPlace.manager != nil {
		panic("progplace.Setup already called")
	}
	myPlace.startConfig = startConfig.Clone()
	myPlace.manager = manager
	myPlace.indexer = idx
}

func (pp *progPlace) Location() string { return "" }

func (pp *progPlace) CanCreateZettel(ctx context.Context) bool { return false }

func (pp *progPlace) CreateZettel(
	ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	return id.Invalid, place.ErrReadOnly
}

func (pp *progPlace) GetZettel(
	ctx context.Context, zid id.Zid) (domain.Zettel, error) {
	if gen, ok := pp.zettel[zid]; ok && gen.meta != nil {
		if m := gen.meta(zid); m != nil {
			updateMeta(m)
			if genContent := gen.content; genContent != nil {
				return domain.Zettel{
					Meta:    m,
					Content: domain.NewContent(genContent(m)),
				}, nil
			}
			return domain.Zettel{Meta: m}, nil
		}
	}
	return domain.Zettel{}, place.ErrNotFound
}

func (pp *progPlace) GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	if gen, ok := pp.zettel[zid]; ok {
		if genMeta := gen.meta; genMeta != nil {
			if m := genMeta(zid); m != nil {
				updateMeta(m)
				return m, nil
			}
		}
	}
	return nil, place.ErrNotFound
}

func (pp *progPlace) FetchZids(ctx context.Context) (map[id.Zid]bool, error) {
	result := make(map[id.Zid]bool, len(pp.zettel))
	for zid, gen := range pp.zettel {
		if genMeta := gen.meta; genMeta != nil {
			if genMeta(zid) != nil {
				result[zid] = true
			}
		}
	}
	return result, nil
}

func (pp *progPlace) SelectMeta(
	ctx context.Context, f *place.Filter, s *place.Sorter) (res []*meta.Meta, err error) {
	hasMatch := place.CreateFilterFunc(f)
	for zid, gen := range pp.zettel {
		if genMeta := gen.meta; genMeta != nil {
			if m := genMeta(zid); m != nil {
				updateMeta(m)
				pp.filter.Update(ctx, m)
				if hasMatch(m) {
					res = append(res, m)
				}
			}
		}
	}
	return place.ApplySorter(res, s), nil
}

func (pp *progPlace) CanUpdateZettel(ctx context.Context, zettel domain.Zettel) bool {
	return false
}

func (pp *progPlace) UpdateZettel(ctx context.Context, zettel domain.Zettel) error {
	return place.ErrReadOnly
}

func (pp *progPlace) AllowRenameZettel(ctx context.Context, zid id.Zid) bool {
	_, ok := pp.zettel[zid]
	return !ok
}

func (pp *progPlace) RenameZettel(ctx context.Context, curZid, newZid id.Zid) error {
	if _, ok := pp.zettel[curZid]; ok {
		return place.ErrReadOnly
	}
	return place.ErrNotFound
}

func (pp *progPlace) CanDeleteZettel(ctx context.Context, zid id.Zid) bool { return false }

func (pp *progPlace) DeleteZettel(ctx context.Context, zid id.Zid) error {
	if _, ok := pp.zettel[zid]; ok {
		return place.ErrReadOnly
	}
	return place.ErrNotFound
}

func (pp *progPlace) Reload(ctx context.Context) error { return nil }

func (pp *progPlace) ReadStats(st *place.Stats) {
	st.ReadOnly = true
	st.Zettel = len(pp.zettel)
}

func updateMeta(m *meta.Meta) {
	m.Set(meta.KeySyntax, meta.ValueSyntaxZmk)
	m.Set(meta.KeyRole, meta.ValueRoleConfiguration)
	m.Set(meta.KeyReadOnly, meta.ValueTrue)
	if _, ok := m.Get(meta.KeyVisibility); !ok {
		m.Set(meta.KeyVisibility, meta.ValueVisibilityExpert)
	}
}
