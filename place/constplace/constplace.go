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
	_ "embed" // Allow to embed file content
	"net/url"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/index"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/manager"
	"zettelstore.de/z/search"
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

func (cp *constPlace) CreateZettel(ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	return id.Invalid, place.ErrReadOnly
}

func (cp *constPlace) GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error) {
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

func (cp *constPlace) FetchZids(ctx context.Context) (id.Set, error) {
	result := id.NewSetCap(len(cp.zettel))
	for zid := range cp.zettel {
		result[zid] = true
	}
	return result, nil
}

func (cp *constPlace) SelectMeta(
	ctx context.Context, f *search.Filter, s *search.Sorter) (res []*meta.Meta, err error) {

	for zid, zettel := range cp.zettel {
		m := makeMeta(zid, zettel.header)
		cp.filter.Enrich(ctx, m)
		if f.Match(m) {
			res = append(res, m)
		}
	}
	return s.Sort(res), nil
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

func (cp *constPlace) ReadStats(st *place.Stats) {
	st.ReadOnly = true
	st.Zettel = len(cp.zettel)
}

const syntaxTemplate = "mustache"

var constZettelMap = map[id.Zid]constZettel{
	id.ConfigurationZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Runtime Configuration",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityOwner,
			meta.KeySyntax:     meta.ValueSyntaxNone,
		},
		""},
	id.BaseTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Base HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		domain.NewContent(contentBaseMustache)},
	id.LoginTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Login Form HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		domain.NewContent(contentLoginMustache)},
	id.ZettelTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Zettel HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		domain.NewContent(contentZettelMustache)},
	id.InfoTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Info HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		domain.NewContent(contentInfoMustache)},
	id.ContextTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Context HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		domain.NewContent(contentContextMustache)},
	id.FormTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Form HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		domain.NewContent(contentFormMustache)},
	id.RenameTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Rename Form HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		domain.NewContent(contentRenameMustache)},
	id.DeleteTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Delete HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		domain.NewContent(contentDeleteMustache)},
	id.ListTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore List Zettel HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		domain.NewContent(contentListZettelMustache)},
	id.RolesTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore List Roles HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		domain.NewContent(contentListRolesMustache)},
	id.TagsTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore List Tags HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		domain.NewContent(contentListTagsMustache)},
	id.ErrorTemplateZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Error HTML Template",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityExpert,
			meta.KeySyntax:     syntaxTemplate,
		},
		domain.NewContent(contentErrorMustache)},
	id.BaseCSSZid: {
		constHeader{
			meta.KeyTitle:      "Zettelstore Base CSS",
			meta.KeyRole:       meta.ValueRoleConfiguration,
			meta.KeyVisibility: meta.ValueVisibilityPublic,
			meta.KeySyntax:     "css",
		},
		domain.NewContent(contentBaseCSS)},
	id.TOCNewTemplateZid: {
		constHeader{
			meta.KeyTitle:  "New Menu",
			meta.KeyRole:   meta.ValueRoleConfiguration,
			meta.KeySyntax: meta.ValueSyntaxZmk,
		},
		domain.NewContent(contentNewTOCZettel)},
	id.TemplateNewZettelZid: {
		constHeader{
			meta.KeyTitle:  "New Zettel",
			meta.KeyRole:   meta.ValueRoleZettel,
			meta.KeySyntax: meta.ValueSyntaxZmk,
		},
		""},
	id.TemplateNewUserZid: {
		constHeader{
			meta.KeyTitle:      "New User",
			meta.KeyRole:       meta.ValueRoleUser,
			meta.KeyCredential: "",
			meta.KeyUserID:     "",
			meta.KeyUserRole:   meta.ValueUserRoleReader,
			meta.KeySyntax:     meta.ValueSyntaxNone,
		},
		""},
	id.DefaultHomeZid: {
		constHeader{
			meta.KeyTitle:  "Home",
			meta.KeyRole:   meta.ValueRoleZettel,
			meta.KeySyntax: meta.ValueSyntaxZmk,
		},
		domain.NewContent(contentHomeZettel)},
}

//go:embed base.mustache
var contentBaseMustache string

//go:embed login.mustache
var contentLoginMustache string

//go:embed zettel.mustache
var contentZettelMustache string

//go:embed info.mustache
var contentInfoMustache string

//go:embed context.mustache
var contentContextMustache string

//go:embed form.mustache
var contentFormMustache string

//go:embed rename.mustache
var contentRenameMustache string

//go:embed delete.mustache
var contentDeleteMustache string

//go:embed listzettel.mustache
var contentListZettelMustache string

//go:embed listroles.mustache
var contentListRolesMustache string

//go:embed listtags.mustache
var contentListTagsMustache string

//go:embed error.mustache
var contentErrorMustache string

//go:embed base.css
var contentBaseCSS string

//go:embed newtoc.zettel
var contentNewTOCZettel string

//go:embed home.zettel
var contentHomeZettel string
