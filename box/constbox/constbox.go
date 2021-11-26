//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
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
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

func init() {
	manager.Register(
		" const",
		func(u *url.URL, cdata *manager.ConnectData) (box.ManagedBox, error) {
			return &constBox{
				number:   cdata.Number,
				zettel:   constZettelMap,
				enricher: cdata.Enricher,
			}, nil
		})
}

type constHeader map[string]string

type constZettel struct {
	header  constHeader
	content domain.Content
}

type constBox struct {
	number   int
	zettel   map[id.Zid]constZettel
	enricher box.Enricher
}

func (*constBox) Location() string { return "const:" }

func (*constBox) CanCreateZettel(context.Context) bool { return false }

func (*constBox) CreateZettel(context.Context, domain.Zettel) (id.Zid, error) {
	return id.Invalid, box.ErrReadOnly
}

func (cp *constBox) GetZettel(_ context.Context, zid id.Zid) (domain.Zettel, error) {
	if z, ok := cp.zettel[zid]; ok {
		return domain.Zettel{Meta: meta.NewWithData(zid, z.header), Content: z.content}, nil
	}
	return domain.Zettel{}, box.ErrNotFound
}

func (cp *constBox) GetMeta(_ context.Context, zid id.Zid) (*meta.Meta, error) {
	if z, ok := cp.zettel[zid]; ok {
		return meta.NewWithData(zid, z.header), nil
	}
	return nil, box.ErrNotFound
}

func (cp *constBox) ApplyZid(_ context.Context, handle box.ZidFunc) error {
	for zid := range cp.zettel {
		handle(zid)
	}
	return nil
}

func (cp *constBox) ApplyMeta(ctx context.Context, handle box.MetaFunc) error {
	for zid, zettel := range cp.zettel {
		m := meta.NewWithData(zid, zettel.header)
		cp.enricher.Enrich(ctx, m, cp.number)
		handle(m)
	}
	return nil
}

func (*constBox) CanUpdateZettel(context.Context, domain.Zettel) bool { return false }

func (*constBox) UpdateZettel(context.Context, domain.Zettel) error { return box.ErrReadOnly }

func (cp *constBox) AllowRenameZettel(_ context.Context, zid id.Zid) bool {
	_, ok := cp.zettel[zid]
	return !ok
}

func (cp *constBox) RenameZettel(_ context.Context, curZid, _ id.Zid) error {
	if _, ok := cp.zettel[curZid]; ok {
		return box.ErrReadOnly
	}
	return box.ErrNotFound
}
func (*constBox) CanDeleteZettel(context.Context, id.Zid) bool { return false }

func (cp *constBox) DeleteZettel(_ context.Context, zid id.Zid) error {
	if _, ok := cp.zettel[zid]; ok {
		return box.ErrReadOnly
	}
	return box.ErrNotFound
}

func (cp *constBox) ReadStats(st *box.ManagedBoxStats) {
	st.ReadOnly = true
	st.Zettel = len(cp.zettel)
}

const syntaxTemplate = "mustache"

var constZettelMap = map[id.Zid]constZettel{
	id.ConfigurationZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Runtime Configuration",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     api.ValueSyntaxNone,
			api.KeyNoIndex:    api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityOwner,
		},
		domain.NewContent(nil)},
	id.MustParse(api.ZidLicense): {
		constHeader{
			api.KeyTitle:      "Zettelstore License",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     api.ValueSyntaxText,
			api.KeyLang:       api.ValueLangEN,
			api.KeyReadOnly:   api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityPublic,
		},
		domain.NewContent(contentLicense)},
	id.MustParse(api.ZidAuthors): {
		constHeader{
			api.KeyTitle:      "Zettelstore Contributors",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     api.ValueSyntaxZmk,
			api.KeyLang:       api.ValueLangEN,
			api.KeyReadOnly:   api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityLogin,
		},
		domain.NewContent(contentContributors)},
	id.MustParse(api.ZidDependencies): {
		constHeader{
			api.KeyTitle:      "Zettelstore Dependencies",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     api.ValueSyntaxZmk,
			api.KeyLang:       api.ValueLangEN,
			api.KeyReadOnly:   api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityLogin,
		},
		domain.NewContent(contentDependencies)},
	id.BaseTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Base HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     syntaxTemplate,
			api.KeyNoIndex:    api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		domain.NewContent(contentBaseMustache)},
	id.LoginTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Login Form HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     syntaxTemplate,
			api.KeyNoIndex:    api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		domain.NewContent(contentLoginMustache)},
	id.ZettelTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Zettel HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     syntaxTemplate,
			api.KeyNoIndex:    api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		domain.NewContent(contentZettelMustache)},
	id.InfoTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Info HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     syntaxTemplate,
			api.KeyNoIndex:    api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		domain.NewContent(contentInfoMustache)},
	id.ContextTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Context HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     syntaxTemplate,
			api.KeyNoIndex:    api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		domain.NewContent(contentContextMustache)},
	id.FormTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Form HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     syntaxTemplate,
			api.KeyNoIndex:    api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		domain.NewContent(contentFormMustache)},
	id.RenameTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Rename Form HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     syntaxTemplate,
			api.KeyNoIndex:    api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		domain.NewContent(contentRenameMustache)},
	id.DeleteTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Delete HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     syntaxTemplate,
			api.KeyNoIndex:    api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		domain.NewContent(contentDeleteMustache)},
	id.ListTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore List Zettel HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     syntaxTemplate,
			api.KeyNoIndex:    api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		domain.NewContent(contentListZettelMustache)},
	id.RolesTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore List Roles HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     syntaxTemplate,
			api.KeyNoIndex:    api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		domain.NewContent(contentListRolesMustache)},
	id.TagsTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore List Tags HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     syntaxTemplate,
			api.KeyNoIndex:    api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		domain.NewContent(contentListTagsMustache)},
	id.ErrorTemplateZid: {
		constHeader{
			api.KeyTitle:      "Zettelstore Error HTML Template",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     syntaxTemplate,
			api.KeyNoIndex:    api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityExpert,
		},
		domain.NewContent(contentErrorMustache)},
	id.MustParse(api.ZidBaseCSS): {
		constHeader{
			api.KeyTitle:      "Zettelstore Base CSS",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     "css",
			api.KeyNoIndex:    api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityPublic,
		},
		domain.NewContent(contentBaseCSS)},
	id.MustParse(api.ZidUserCSS): {
		constHeader{
			api.KeyTitle:      "Zettelstore User CSS",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     "css",
			api.KeyVisibility: api.ValueVisibilityPublic,
		},
		domain.NewContent([]byte("/* User-defined CSS */"))},
	id.EmojiZid: {
		constHeader{
			api.KeyTitle:      "Generic Emoji",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     api.ValueSyntaxGif,
			api.KeyReadOnly:   api.ValueTrue,
			api.KeyVisibility: api.ValueVisibilityPublic,
		},
		domain.NewContent(contentEmoji)},
	id.TOCNewTemplateZid: {
		constHeader{
			api.KeyTitle:      "New Menu",
			api.KeyRole:       api.ValueRoleConfiguration,
			api.KeySyntax:     api.ValueSyntaxZmk,
			api.KeyLang:       api.ValueLangEN,
			api.KeyVisibility: api.ValueVisibilityCreator,
		},
		domain.NewContent(contentNewTOCZettel)},
	id.MustParse(api.ZidTemplateNewZettel): {
		constHeader{
			api.KeyTitle:      "New Zettel",
			api.KeyRole:       api.ValueRoleZettel,
			api.KeySyntax:     api.ValueSyntaxZmk,
			api.KeyVisibility: api.ValueVisibilityCreator,
		},
		domain.NewContent(nil)},
	id.MustParse(api.ZidTemplateNewUser): {
		constHeader{
			api.KeyTitle:                       "New User",
			api.KeyRole:                        api.ValueRoleUser,
			api.KeySyntax:                      api.ValueSyntaxNone,
			meta.NewPrefix + api.KeyCredential: "",
			meta.NewPrefix + api.KeyUserID:     "",
			meta.NewPrefix + api.KeyUserRole:   api.ValueUserRoleReader,
			api.KeyVisibility:                  api.ValueVisibilityOwner,
		},
		domain.NewContent(nil)},
	id.DefaultHomeZid: {
		constHeader{
			api.KeyTitle:  "Home",
			api.KeyRole:   api.ValueRoleZettel,
			api.KeySyntax: api.ValueSyntaxZmk,
			api.KeyLang:   api.ValueLangEN,
		},
		domain.NewContent(contentHomeZettel)},
}

//go:embed license.txt
var contentLicense []byte

//go:embed contributors.zettel
var contentContributors []byte

//go:embed dependencies.zettel
var contentDependencies []byte

//go:embed base.mustache
var contentBaseMustache []byte

//go:embed login.mustache
var contentLoginMustache []byte

//go:embed zettel.mustache
var contentZettelMustache []byte

//go:embed info.mustache
var contentInfoMustache []byte

//go:embed context.mustache
var contentContextMustache []byte

//go:embed form.mustache
var contentFormMustache []byte

//go:embed rename.mustache
var contentRenameMustache []byte

//go:embed delete.mustache
var contentDeleteMustache []byte

//go:embed listzettel.mustache
var contentListZettelMustache []byte

//go:embed listroles.mustache
var contentListRolesMustache []byte

//go:embed listtags.mustache
var contentListTagsMustache []byte

//go:embed error.mustache
var contentErrorMustache []byte

//go:embed base.css
var contentBaseCSS []byte

//go:embed emoji_spin.gif
var contentEmoji []byte

//go:embed newtoc.zettel
var contentNewTOCZettel []byte

//go:embed home.zettel
var contentHomeZettel []byte
