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
	"sync"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/manager"
	"zettelstore.de/z/search"
)

func init() {
	manager.Register(
		" prog",
		func(u *url.URL, cdata *manager.ConnectData) (place.ManagedPlace, error) {
			return getPlace(cdata.Enricher), nil
		})
}

type (
	zettelGen struct {
		meta    func(id.Zid) *meta.Meta
		content func(*meta.Meta) string
	}

	progPlace struct {
		zettel      map[id.Zid]zettelGen
		filter      place.Enricher
		startConfig *meta.Meta
	}
)

var (
	myPlace *progPlace
	myMx    sync.RWMutex
)

// Get returns the one program place.
func getPlace(mf place.Enricher) place.ManagedPlace {
	myMx.Lock()
	if myPlace == nil {
		myPlace = &progPlace{
			zettel: map[id.Zid]zettelGen{
				id.VersionZid:              {genVersionBuildM, genVersionBuildC},
				id.HostZid:                 {genVersionHostM, genVersionHostC},
				id.OperatingSystemZid:      {genVersionOSM, genVersionOSC},
				id.PlaceManagerZid:         {genManagerM, genManagerC},
				id.MetadataKeyZid:          {genKeysM, genKeysC},
				id.StartupConfigurationZid: {genConfigZettelM, genConfigZettelC},
			},
			filter: mf,
		}
	}
	myMx.Unlock()
	return myPlace
}

// Setup remembers important values.
func Setup(startConfig *meta.Meta) {
	myMx.Lock()
	defer myMx.Unlock()
	if myPlace == nil {
		panic("progplace.getPlace not called")
	}
	if myPlace.startConfig != nil {
		panic("progplace.Setup already called")
	}
	myPlace.startConfig = startConfig.Clone()
}

func (pp *progPlace) Location() string { return "" }

func (pp *progPlace) CanCreateZettel(ctx context.Context) bool { return false }

func (pp *progPlace) CreateZettel(
	ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	return id.Invalid, place.ErrReadOnly
}

func (pp *progPlace) GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error) {
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

func (pp *progPlace) FetchZids(ctx context.Context) (id.Set, error) {
	result := id.NewSetCap(len(pp.zettel))
	for zid, gen := range pp.zettel {
		if genMeta := gen.meta; genMeta != nil {
			if genMeta(zid) != nil {
				result[zid] = true
			}
		}
	}
	return result, nil
}

func (pp *progPlace) SelectMeta(ctx context.Context, match search.MetaMatchFunc) (res []*meta.Meta, err error) {
	for zid, gen := range pp.zettel {
		if genMeta := gen.meta; genMeta != nil {
			if m := genMeta(zid); m != nil {
				updateMeta(m)
				pp.filter.Enrich(ctx, m)
				if match(m) {
					res = append(res, m)
				}
			}
		}
	}
	return res, nil
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

func (pp *progPlace) ReadStats(st *place.ManagedPlaceStats) {
	st.ReadOnly = true
	st.Zettel = len(pp.zettel)
}

func updateMeta(m *meta.Meta) {
	m.Set(meta.KeyNoIndex, meta.ValueTrue)
	m.Set(meta.KeySyntax, meta.ValueSyntaxZmk)
	m.Set(meta.KeyRole, meta.ValueRoleConfiguration)
	m.Set(meta.KeyLang, meta.ValueLangEN)
	m.Set(meta.KeyReadOnly, meta.ValueTrue)
	if _, ok := m.Get(meta.KeyVisibility); !ok {
		m.Set(meta.KeyVisibility, meta.ValueVisibilityExpert)
	}
}
