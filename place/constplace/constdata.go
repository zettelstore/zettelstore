//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package constplace stores zettel inside the executable.
package constplace

import (
	_ "embed" // Allow to embed file content

	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

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

//go:embed base.css
var contentBaseCSS string

//go:embed newtoc.zettel
var contentNewTOCZettel string

//go:embed home.zettel
var contentHomeZettel string
