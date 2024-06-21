//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2020-present Detlef Stern
//-----------------------------------------------------------------------------

// Package compbox provides zettel that have computed content.
package compbox

import (
	"context"
	"net/url"

	"t73f.de/r/zsc/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/box/manager"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/logger"
	"zettelstore.de/z/query"
	"zettelstore.de/z/zettel"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
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
var myZettel = map[id.ZidO]struct {
	meta    func(id.ZidO) *meta.Meta
	content func(*meta.Meta) []byte
}{
	id.MustParseO(api.ZidVersion):              {genVersionBuildM, genVersionBuildC},
	id.MustParseO(api.ZidHost):                 {genVersionHostM, genVersionHostC},
	id.MustParseO(api.ZidOperatingSystem):      {genVersionOSM, genVersionOSC},
	id.MustParseO(api.ZidLog):                  {genLogM, genLogC},
	id.MustParseO(api.ZidMemory):               {genMemoryM, genMemoryC},
	id.MustParseO(api.ZidBoxManager):           {genManagerM, genManagerC},
	id.MustParseO(api.ZidMetadataKey):          {genKeysM, genKeysC},
	id.MustParseO(api.ZidParser):               {genParserM, genParserC},
	id.MustParseO(api.ZidStartupConfiguration): {genConfigZettelM, genConfigZettelC},
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

func (cb *compBox) GetZettel(_ context.Context, zid id.ZidO) (zettel.Zettel, error) {
	if gen, ok := myZettel[zid]; ok && gen.meta != nil {
		if m := gen.meta(zid); m != nil {
			updateMeta(m)
			if genContent := gen.content; genContent != nil {
				cb.log.Trace().Msg("GetZettel/Content")
				return zettel.Zettel{
					Meta:    m,
					Content: zettel.NewContent(genContent(m)),
				}, nil
			}
			cb.log.Trace().Msg("GetZettel/NoContent")
			return zettel.Zettel{Meta: m}, nil
		}
	}
	err := box.ErrZettelNotFound{Zid: zid}
	cb.log.Trace().Err(err).Msg("GetZettel/Err")
	return zettel.Zettel{}, err
}

func (*compBox) HasZettel(_ context.Context, zid id.ZidO) bool {
	_, found := myZettel[zid]
	return found
}

func (cb *compBox) ApplyZid(_ context.Context, handle box.ZidFunc, constraint query.RetrievePredicate) error {
	cb.log.Trace().Int("entries", int64(len(myZettel))).Msg("ApplyZid")
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

func (*compBox) AllowRenameZettel(_ context.Context, zid id.ZidO) bool {
	_, ok := myZettel[zid]
	return !ok
}

func (cb *compBox) RenameZettel(_ context.Context, curZid, _ id.ZidO) (err error) {
	if _, ok := myZettel[curZid]; ok {
		err = box.ErrReadOnly
	} else {
		err = box.ErrZettelNotFound{Zid: curZid}
	}
	cb.log.Trace().Err(err).Msg("RenameZettel")
	return err
}

func (*compBox) CanDeleteZettel(context.Context, id.ZidO) bool { return false }

func (cb *compBox) DeleteZettel(_ context.Context, zid id.ZidO) (err error) {
	if _, ok := myZettel[zid]; ok {
		err = box.ErrReadOnly
	} else {
		err = box.ErrZettelNotFound{Zid: zid}
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
