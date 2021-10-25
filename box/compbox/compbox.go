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

	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/box/manager"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
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
	content func(*meta.Meta) []byte
}{
	id.MustParse(api.ZidVersion):              {genVersionBuildM, genVersionBuildC},
	id.MustParse(api.ZidHost):                 {genVersionHostM, genVersionHostC},
	id.MustParse(api.ZidOperatingSystem):      {genVersionOSM, genVersionOSC},
	id.MustParse(api.ZidBoxManager):           {genManagerM, genManagerC},
	id.MustParse(api.ZidMetadataKey):          {genKeysM, genKeysC},
	id.MustParse(api.ZidStartupConfiguration): {genConfigZettelM, genConfigZettelC},
}

// Get returns the one program box.
func getCompBox(boxNumber int, mf box.Enricher) box.ManagedBox {
	return &compBox{number: boxNumber, enricher: mf}
}

// Setup remembers important values.
func Setup(cfg *meta.Meta) { myConfig = cfg.Clone() }

func (*compBox) Location() string { return "" }

func (*compBox) CanCreateZettel(context.Context) bool { return false }

func (*compBox) CreateZettel(context.Context, domain.Zettel) (id.Zid, error) {
	return id.Invalid, box.ErrReadOnly
}

func (*compBox) GetZettel(_ context.Context, zid id.Zid) (domain.Zettel, error) {
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

func (*compBox) GetMeta(_ context.Context, zid id.Zid) (*meta.Meta, error) {
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

func (*compBox) ApplyZid(_ context.Context, handle box.ZidFunc) error {
	for zid, gen := range myZettel {
		if genMeta := gen.meta; genMeta != nil {
			if genMeta(zid) != nil {
				handle(zid)
			}
		}
	}
	return nil
}

func (pp *compBox) ApplyMeta(ctx context.Context, handle box.MetaFunc) error {
	for zid, gen := range myZettel {
		if genMeta := gen.meta; genMeta != nil {
			if m := genMeta(zid); m != nil {
				updateMeta(m)
				pp.enricher.Enrich(ctx, m, pp.number)
				handle(m)
			}
		}
	}
	return nil
}

func (*compBox) CanUpdateZettel(context.Context, domain.Zettel) bool { return false }

func (*compBox) UpdateZettel(context.Context, domain.Zettel) error { return box.ErrReadOnly }

func (*compBox) AllowRenameZettel(_ context.Context, zid id.Zid) bool {
	_, ok := myZettel[zid]
	return !ok
}

func (*compBox) RenameZettel(_ context.Context, curZid, _ id.Zid) error {
	if _, ok := myZettel[curZid]; ok {
		return box.ErrReadOnly
	}
	return box.ErrNotFound
}

func (*compBox) CanDeleteZettel(context.Context, id.Zid) bool { return false }

func (*compBox) DeleteZettel(_ context.Context, zid id.Zid) error {
	if _, ok := myZettel[zid]; ok {
		return box.ErrReadOnly
	}
	return box.ErrNotFound
}

func (*compBox) ReadStats(st *box.ManagedBoxStats) {
	st.ReadOnly = true
	st.Zettel = len(myZettel)
}

func updateMeta(m *meta.Meta) {
	m.Set(api.KeyNoIndex, api.ValueTrue)
	m.Set(api.KeySyntax, api.ValueSyntaxZmk)
	m.Set(api.KeyRole, api.ValueRoleConfiguration)
	m.Set(api.KeyLang, api.ValueLangEN)
	m.Set(api.KeyReadOnly, api.ValueTrue)
	if _, ok := m.Get(api.KeyVisibility); !ok {
		m.Set(api.KeyVisibility, api.ValueVisibilityExpert)
	}
}
