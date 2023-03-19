//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
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
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/logger"
	"zettelstore.de/z/query"
)

func init() {
	manager.Register(
		" comp",
		func(u *url.URL, cdata *manager.ConnectData) (box.ManagedBox, error) {
			return getCompBox(cdata.Number, cdata.Enricher), nil
		})
}

type compBox struct {
	log      *logger.Logger
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
	id.MustParse(api.ZidLog):                  {genLogM, genLogC},
	id.MustParse(api.ZidBoxManager):           {genManagerM, genManagerC},
	id.MustParse(api.ZidMetadataKey):          {genKeysM, genKeysC},
	id.MustParse(api.ZidParser):               {genParserM, genParserC},
	id.MustParse(api.ZidStartupConfiguration): {genConfigZettelM, genConfigZettelC},
}

// Get returns the one program box.
func getCompBox(boxNumber int, mf box.Enricher) box.ManagedBox {
	return &compBox{
		log: kernel.Main.GetLogger(kernel.BoxService).Clone().
			Str("box", "comp").Int("boxnum", int64(boxNumber)).Child(),
		number:   boxNumber,
		enricher: mf,
	}
}

// Setup remembers important values.
func Setup(cfg *meta.Meta) { myConfig = cfg.Clone() }

func (*compBox) Location() string { return "" }

func (*compBox) CanCreateZettel(context.Context) bool { return false }

func (cb *compBox) CreateZettel(context.Context, domain.Zettel) (id.Zid, error) {
	cb.log.Trace().Err(box.ErrReadOnly).Msg("CreateZettel")
	return id.Invalid, box.ErrReadOnly
}

func (cb *compBox) GetZettel(_ context.Context, zid id.Zid) (domain.Zettel, error) {
	if gen, ok := myZettel[zid]; ok && gen.meta != nil {
		if m := gen.meta(zid); m != nil {
			updateMeta(m)
			if genContent := gen.content; genContent != nil {
				cb.log.Trace().Msg("GetMeta/Content")
				return domain.Zettel{
					Meta:    m,
					Content: domain.NewContent(genContent(m)),
				}, nil
			}
			cb.log.Trace().Msg("GetMeta/NoContent")
			return domain.Zettel{Meta: m}, nil
		}
	}
	cb.log.Trace().Err(box.ErrNotFound).Msg("GetZettel/Err")
	return domain.Zettel{}, box.ErrNotFound
}

func (cb *compBox) GetMeta(_ context.Context, zid id.Zid) (*meta.Meta, error) {
	if gen, ok := myZettel[zid]; ok {
		if genMeta := gen.meta; genMeta != nil {
			if m := genMeta(zid); m != nil {
				updateMeta(m)
				cb.log.Trace().Msg("GetMeta")
				return m, nil
			}
		}
	}
	cb.log.Trace().Err(box.ErrNotFound).Msg("GetMeta/Err")
	return nil, box.ErrNotFound
}

func (cb *compBox) ApplyZid(_ context.Context, handle box.ZidFunc, constraint query.RetrievePredicate) error {
	cb.log.Trace().Int("entries", int64(len(myZettel))).Msg("ApplyMeta")
	for zid, gen := range myZettel {
		if !constraint(zid) {
			continue
		}
		if genMeta := gen.meta; genMeta != nil {
			if genMeta(zid) != nil {
				handle(zid)
			}
		}
	}
	return nil
}

func (cb *compBox) ApplyMeta(ctx context.Context, handle box.MetaFunc, constraint query.RetrievePredicate) error {
	cb.log.Trace().Int("entries", int64(len(myZettel))).Msg("ApplyMeta")
	for zid, gen := range myZettel {
		if !constraint(zid) {
			continue
		}
		if genMeta := gen.meta; genMeta != nil {
			if m := genMeta(zid); m != nil {
				updateMeta(m)
				cb.enricher.Enrich(ctx, m, cb.number)
				handle(m)
			}
		}
	}
	return nil
}

func (*compBox) CanUpdateZettel(context.Context, domain.Zettel) bool { return false }

func (cb *compBox) UpdateZettel(context.Context, domain.Zettel) error {
	cb.log.Trace().Err(box.ErrReadOnly).Msg("UpdateZettel")
	return box.ErrReadOnly
}

func (*compBox) AllowRenameZettel(_ context.Context, zid id.Zid) bool {
	_, ok := myZettel[zid]
	return !ok
}

func (cb *compBox) RenameZettel(_ context.Context, curZid, _ id.Zid) error {
	err := box.ErrNotFound
	if _, ok := myZettel[curZid]; ok {
		err = box.ErrReadOnly
	}
	cb.log.Trace().Err(err).Msg("RenameZettel")
	return err
}

func (*compBox) CanDeleteZettel(context.Context, id.Zid) bool { return false }

func (cb *compBox) DeleteZettel(_ context.Context, zid id.Zid) error {
	err := box.ErrNotFound
	if _, ok := myZettel[zid]; ok {
		err = box.ErrReadOnly
	}
	cb.log.Trace().Err(err).Msg("DeleteZettel")
	return err
}

func (cb *compBox) ReadStats(st *box.ManagedBoxStats) {
	st.ReadOnly = true
	st.Zettel = len(myZettel)
	cb.log.Trace().Int("zettel", int64(st.Zettel)).Msg("ReadStats")
}

func updateMeta(m *meta.Meta) {
	if _, ok := m.Get(api.KeySyntax); !ok {
		m.Set(api.KeySyntax, meta.SyntaxZmk)
	}
	m.Set(api.KeyRole, api.ValueRoleConfiguration)
	m.Set(api.KeyLang, api.ValueLangEN)
	m.Set(api.KeyReadOnly, api.ValueTrue)
	if _, ok := m.Get(api.KeyVisibility); !ok {
		m.Set(api.KeyVisibility, api.ValueVisibilityExpert)
	}
}
