//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package compbox provides zettel that have computed content.
package compbox

import (
	"context"
	"net/url"

	"zettelstore.de/z/box"
	"zettelstore.de/z/box/manager"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/search"
)

func init() {
	manager.Register(
		" comp",
		func(u *url.URL, cdata *manager.ConnectData) (box.ManagedBox, error) {
			return getCompBox(cdata.Number, cdata.Enricher), nil
		})
}

type compBox struct {
	number   int
	enricher box.Enricher
}

var myConfig *meta.Meta
var myZettel = map[id.Zid]struct {
	meta    func(id.Zid) *meta.Meta
	content func(*meta.Meta) string
}{
	id.VersionZid:              {genVersionBuildM, genVersionBuildC},
	id.HostZid:                 {genVersionHostM, genVersionHostC},
	id.OperatingSystemZid:      {genVersionOSM, genVersionOSC},
	id.BoxManagerZid:           {genManagerM, genManagerC},
	id.MetadataKeyZid:          {genKeysM, genKeysC},
	id.StartupConfigurationZid: {genConfigZettelM, genConfigZettelC},
}

// Get returns the one program box.
func getCompBox(boxNumber int, mf box.Enricher) box.ManagedBox {
	return &compBox{number: boxNumber, enricher: mf}
}

// Setup remembers important values.
func Setup(cfg *meta.Meta) { myConfig = cfg.Clone() }

func (pp *compBox) Location() string { return "" }

func (pp *compBox) CanCreateZettel(ctx context.Context) bool { return false }

func (pp *compBox) CreateZettel(
	ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	return id.Invalid, box.ErrReadOnly
}

func (pp *compBox) GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error) {
	if gen, ok := myZettel[zid]; ok && gen.meta != nil {
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
	return domain.Zettel{}, box.ErrNotFound
}

func (pp *compBox) GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error) {
	if gen, ok := myZettel[zid]; ok {
		if genMeta := gen.meta; genMeta != nil {
			if m := genMeta(zid); m != nil {
				updateMeta(m)
				return m, nil
			}
		}
	}
	return nil, box.ErrNotFound
}

func (pp *compBox) FetchZids(ctx context.Context) (id.Set, error) {
	result := id.NewSetCap(len(myZettel))
	for zid, gen := range myZettel {
		if genMeta := gen.meta; genMeta != nil {
			if genMeta(zid) != nil {
				result[zid] = true
			}
		}
	}
	return result, nil
}

func (pp *compBox) SelectMeta(ctx context.Context, match search.MetaMatchFunc) (sel, rej box.MetaMap, err error) {
	sel = make(box.MetaMap, len(myZettel))
	rej = make(box.MetaMap, len(myZettel))
	for zid, gen := range myZettel {
		if genMeta := gen.meta; genMeta != nil {
			if m := genMeta(zid); m != nil {
				updateMeta(m)
				pp.enricher.Enrich(ctx, m, pp.number)
				if match(m) {
					sel[zid] = m
				} else {
					rej[zid] = m
				}
			}
		}
	}
	return sel, nil, nil
}

func (pp *compBox) CanUpdateZettel(ctx context.Context, zettel domain.Zettel) bool {
	return false
}

func (pp *compBox) UpdateZettel(ctx context.Context, zettel domain.Zettel) error {
	return box.ErrReadOnly
}

func (pp *compBox) AllowRenameZettel(ctx context.Context, zid id.Zid) bool {
	_, ok := myZettel[zid]
	return !ok
}

func (pp *compBox) RenameZettel(ctx context.Context, curZid, newZid id.Zid) error {
	if _, ok := myZettel[curZid]; ok {
		return box.ErrReadOnly
	}
	return box.ErrNotFound
}

func (pp *compBox) CanDeleteZettel(ctx context.Context, zid id.Zid) bool { return false }

func (pp *compBox) DeleteZettel(ctx context.Context, zid id.Zid) error {
	if _, ok := myZettel[zid]; ok {
		return box.ErrReadOnly
	}
	return box.ErrNotFound
}

func (pp *compBox) ReadStats(st *box.ManagedBoxStats) {
	st.ReadOnly = true
	st.Zettel = len(myZettel)
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
