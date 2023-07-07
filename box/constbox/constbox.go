//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package constbox puts zettel inside the executable.
package constbox

import (
	"context"
	_ "embed" // Allow to embed file content
	"net/url"

	"zettelstore.de/c/api"
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
		" const",
		func(u *url.URL, cdata *manager.ConnectData) (box.ManagedBox, error) {
			return &constBox{
				log: kernel.Main.GetLogger(kernel.BoxService).Clone().
					Str("box", "const").Int("boxnum", int64(cdata.Number)).Child(),
				number:   cdata.Number,
				zettel:   constZettelMap,
				enricher: cdata.Enricher,
			}, nil
		})
}

type constHeader map[string]string

type constZettel struct {
	header  constHeader
	content zettel.Content
}

type constBox struct {
	log      *logger.Logger
	number   int
	zettel   map[id.Zid]constZettel
	enricher box.Enricher
}

func (*constBox) Location() string { return "const:" }

func (*constBox) CanCreateZettel(context.Context) bool { return false }

func (cb *constBox) CreateZettel(context.Context, zettel.Zettel) (id.Zid, error) {
	cb.log.Trace().Err(box.ErrReadOnly).Msg("CreateZettel")
	return id.Invalid, box.ErrReadOnly
}

func (cb *constBox) GetZettel(_ context.Context, zid id.Zid) (zettel.Zettel, error) {
	if z, ok := cb.zettel[zid]; ok {
		cb.log.Trace().Msg("GetZettel")
		return zettel.Zettel{Meta: meta.NewWithData(zid, z.header), Content: z.content}, nil
	}
	cb.log.Trace().Err(box.ErrNotFound).Msg("GetZettel")
	return zettel.Zettel{}, box.ErrNotFound
}

func (cb *constBox) GetMeta(_ context.Context, zid id.Zid) (*meta.Meta, error) {
	if z, ok := cb.zettel[zid]; ok {
		cb.log.Trace().Msg("GetMeta")
		return meta.NewWithData(zid, z.header), nil
	}
	cb.log.Trace().Err(box.ErrNotFound).Msg("GetMeta")
	return nil, box.ErrNotFound
}

func (cb *constBox) ApplyZid(_ context.Context, handle box.ZidFunc, constraint query.RetrievePredicate) error {
	cb.log.Trace().Int("entries", int64(len(cb.zettel))).Msg("ApplyZid")
	for zid := range cb.zettel {
		if constraint(zid) {
			handle(zid)
		}
	}
	return nil
}

func (cb *constBox) ApplyMeta(ctx context.Context, handle box.MetaFunc, constraint query.RetrievePredicate) error {
	cb.log.Trace().Int("entries", int64(len(cb.zettel))).Msg("ApplyMeta")
	for zid, zettel := range cb.zettel {
		if constraint(zid) {
			m := meta.NewWithData(zid, zettel.header)
			cb.enricher.Enrich(ctx, m, cb.number)
			handle(m)
		}
	}
	return nil
}

func (*constBox) CanUpdateZettel(context.Context, zettel.Zettel) bool { return false }

func (cb *constBox) UpdateZettel(context.Context, zettel.Zettel) error {
	cb.log.Trace().Err(box.ErrReadOnly).Msg("UpdateZettel")
	return box.ErrReadOnly
}

func (cb *constBox) AllowRenameZettel(_ context.Context, zid id.Zid) bool {
	_, ok := cb.zettel[zid]
	return !ok
}

func (cb *constBox) RenameZettel(_ context.Context, curZid, _ id.Zid) error {
	err := box.ErrNotFound
	if _, ok := cb.zettel[curZid]; ok {
		err = box.ErrReadOnly
	}
	cb.log.Trace().Err(err).Msg("RenameZettel")
	return err
}

func (*constBox) CanDeleteZettel(context.Context, id.Zid) bool { return false }

func (cb *constBox) DeleteZettel(_ context.Context, zid id.Zid) error {
	err := box.ErrNotFound
	if _, ok := cb.zettel[zid]; ok {
		err = box.ErrReadOnly
	}
	cb.log.Trace().Err(err).Msg("DeleteZettel")
	return err
}

func (cb *constBox) ReadStats(st *box.ManagedBoxStats) {
	st.ReadOnly = true
	st.Zettel = len(cb.zettel)
	cb.log.Trace().Int("zettel", int64(st.Zettel)).Msg("ReadStats")
}

var constZettelMap = map[id.Zid]constZettel{
	id.ConfigurationZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Runtime Configuration",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     meta.SyntaxNone,
			api.KeyCreated:    "20200804111624",
			api.KeyVisibility: api.ValueVisibilityOwner,
		},
		zettel.NewContent(nil)},
	id.MustParse(api.ZidLicense): {
		constHeader{
			api.KeyTitle:      "Zettelstore License",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     meta.SyntaxText,
			api.KeyCreated:    "20210504135842",
			api.KeyLang:       api.ValueLangEN,
			api.KeyModified:   "20220131153422",
			api.KeyReadOnly:   api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityPublic,
		},
		zettel.NewContent(contentLicense)},
	id.MustParse(api.ZidAuthors): {
		constHeader{
			api.KeyTitle:      "Zettelstore Contributors",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     meta.SyntaxZmk,
			api.KeyCreated:    "20210504135842",
			api.KeyLang:       api.ValueLangEN,
			api.KeyReadOnly:   api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityLogin,
		},
		zettel.NewContent(contentContributors)},
	id.MustParse(api.ZidDependencies): {
		constHeader{
			api.KeyTitle:      "Zettelstore Dependencies",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     meta.SyntaxZmk,
			api.KeyLang:       api.ValueLangEN,
			api.KeyReadOnly:   api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityLogin,
			api.KeyCreated:    "20210504135842",
			api.KeyModified:   "20230601163100",
		},
		zettel.NewContent(contentDependencies)},
	id.BaseTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Base HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     meta.SyntaxSxn,
			api.KeyCreated:    "20230510155100",
			api.KeyModified:   "20230523144403",
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		zettel.NewContent(contentBaseSxn)},
	id.LoginTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Login Form HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     meta.SyntaxSxn,
			api.KeyCreated:    "20200804111624",
			api.KeyModified:   "20230527144100",
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		zettel.NewContent(contentLoginSxn)},
	id.ZettelTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Zettel HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     meta.SyntaxSxn,
			api.KeyCreated:    "20230510155300",
			api.KeyModified:   "20230523212800",
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		zettel.NewContent(contentZettelSxn)},
	id.InfoTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Info HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     meta.SyntaxSxn,
			api.KeyCreated:    "20200804111624",
			api.KeyModified:   "20230621131500",
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		zettel.NewContent(contentInfoSxn)},
	id.FormTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Form HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     meta.SyntaxSxn,
			api.KeyCreated:    "20200804111624",
			api.KeyModified:   "20230621132600",
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		zettel.NewContent(contentFormSxn)},
	id.RenameTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Rename Form HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     meta.SyntaxSxn,
			api.KeyCreated:    "20200804111624",
			api.KeyModified:   "20230707190246",
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		zettel.NewContent(contentRenameSxn)},
	id.DeleteTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Delete HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     meta.SyntaxSxn,
			api.KeyCreated:    "20200804111624",
			api.KeyModified:   "20230621133100",
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		zettel.NewContent(contentDeleteSxn)},
	id.ListTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore List Zettel HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     meta.SyntaxSxn,
			api.KeyCreated:    "20230704122100",
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		zettel.NewContent(contentListZettelSxn)},
	id.ErrorTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Error HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     meta.SyntaxSxn,
			api.KeyCreated:    "20210305133215",
			api.KeyModified:   "20230527224800",
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		zettel.NewContent(contentErrorSxn)},
	id.TemplateSxnZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Sxn Code for Templates",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     meta.SyntaxSxn,
			api.KeyCreated:    "20230619132800",
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		zettel.NewContent(contentTemplateCodeSxn)},
	id.MustParse(api.ZidBaseCSS): {
		constHeader{
			api.KeyTitle:      "Zettelstore Base CSS",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     meta.SyntaxCSS,
			api.KeyCreated:    "20200804111624",
			api.KeyVisibility: api.ValueVisibilityPublic,
		},
		zettel.NewContent(contentBaseCSS)},
	id.MustParse(api.ZidUserCSS): {
		constHeader{
			api.KeyTitle:      "Zettelstore User CSS",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     meta.SyntaxCSS,
			api.KeyCreated:    "20210622110143",
			api.KeyVisibility: api.ValueVisibilityPublic,
		},
		zettel.NewContent([]byte("/* User-defined CSS */"))},
	id.RoleCSSMapZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Role to CSS Map",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     meta.SyntaxNone,
			api.KeyCreated:    "20220321183214",
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		zettel.NewContent(nil)},
	id.EmojiZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Generic Emoji",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     meta.SyntaxGif,
			api.KeyReadOnly:   api.ValueTrue,
			api.KeyCreated:    "20210504175807",
			api.KeyVisibility: api.ValueVisibilityPublic,
		},
		zettel.NewContent(contentEmoji)},
	id.TOCNewTemplateZid: {
		constHeader{
			api.KeyTitle:      "New Menu",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     meta.SyntaxZmk,
			api.KeyLang:       api.ValueLangEN,
			api.KeyCreated:    "20210217161829",
			api.KeyVisibility: api.ValueVisibilityCreator,
		},
		zettel.NewContent(contentNewTOCZettel)},
	id.MustParse(api.ZidTemplateNewZettel): {
		constHeader{
			api.KeyTitle:      "New Zettel",
			api.KeyRole:       api.ValueRoleZettel,
			api.KeySyntax:     meta.SyntaxZmk,
			api.KeyCreated:    "20201028185209",
			api.KeyVisibility: api.ValueVisibilityCreator,
		},
		zettel.NewContent(nil)},
	id.MustParse(api.ZidTemplateNewUser): {
		constHeader{
			api.KeyTitle:                       "New User",
			api.KeyRole:                        api.ValueRoleConfiguration,
			api.KeySyntax:                      meta.SyntaxNone,
			api.KeyCreated:                     "20201028185209",
			meta.NewPrefix + api.KeyCredential: "",
			meta.NewPrefix + api.KeyUserID:     "",
			meta.NewPrefix + api.KeyUserRole:   api.ValueUserRoleReader,
			api.KeyVisibility:                  api.ValueVisibilityOwner,
		},
		zettel.NewContent(nil)},
	id.DefaultHomeZid: {
		constHeader{
			api.KeyTitle:   "Home",
			api.KeyRole:    api.ValueRoleZettel,
			api.KeySyntax:  meta.SyntaxZmk,
			api.KeyLang:    api.ValueLangEN,
			api.KeyCreated: "20210210190757",
		},
		zettel.NewContent(contentHomeZettel)},
}

//go:embed license.txt
var contentLicense []byte

//go:embed contributors.zettel
var contentContributors []byte

//go:embed dependencies.zettel
var contentDependencies []byte

//go:embed base.sxn
var contentBaseSxn []byte

//go:embed login.sxn
var contentLoginSxn []byte

//go:embed zettel.sxn
var contentZettelSxn []byte

//go:embed info.sxn
var contentInfoSxn []byte

//go:embed form.sxn
var contentFormSxn []byte

//go:embed rename.sxn
var contentRenameSxn []byte

//go:embed delete.sxn
var contentDeleteSxn []byte

//go:embed listzettel.sxn
var contentListZettelSxn []byte

//go:embed error.sxn
var contentErrorSxn []byte

//go:embed wuicode.sxn
var contentTemplateCodeSxn []byte

//go:embed base.css
var contentBaseCSS []byte

//go:embed emoji_spin.gif
var contentEmoji []byte

//go:embed newtoc.zettel
var contentNewTOCZettel []byte

//go:embed home.zettel
var contentHomeZettel []byte
