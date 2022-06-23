//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package usecase

import (
	"context"

	"zettelstore.de/c/api"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/logger"
)

// CreateZettelPort is the interface used by this use case.
type CreateZettelPort interface {
	// CreateZettel creates a new zettel.
	CreateZettel(ctx context.Context, zettel domain.Zettel) (id.Zid, error)
}

// CreateZettel is the data for this use case.
type CreateZettel struct {
	log      *logger.Logger
	rtConfig config.Config
	port     CreateZettelPort
}

// NewCreateZettel creates a new use case.
func NewCreateZettel(log *logger.Logger, rtConfig config.Config, port CreateZettelPort) CreateZettel {
	return CreateZettel{
		log:      log,
		rtConfig: rtConfig,
		port:     port,
	}
}

// PrepareCopy the zettel for further modification.
func (*CreateZettel) PrepareCopy(origZettel domain.Zettel) domain.Zettel {
	m := origZettel.Meta.Clone()
	if title, ok := m.Get(api.KeyTitle); ok {
		m.Set(api.KeyTitle, prependTitle(title, "Copy", "Copy of "))
	}
	if readonly, ok := m.Get(api.KeyReadOnly); ok {
		m.Set(api.KeyReadOnly, copyReadonly(readonly))
	}
	content := origZettel.Content
	content.TrimSpace()
	return domain.Zettel{Meta: m, Content: content}
}

// PrepareFolge the zettel for further modification.
func (uc *CreateZettel) PrepareFolge(origZettel domain.Zettel) domain.Zettel {
	origMeta := origZettel.Meta
	m := meta.New(id.Invalid)
	if title, ok := origMeta.Get(api.KeyTitle); ok {
		m.Set(api.KeyTitle, prependTitle(title, "Folge", "Folge of "))
	}
	m.SetNonEmpty(api.KeyRole, config.GetRole(origMeta, uc.rtConfig))
	m.SetNonEmpty(api.KeyTags, origMeta.GetDefault(api.KeyTags, ""))
	m.SetNonEmpty(api.KeySyntax, uc.rtConfig.GetDefaultSyntax())
	m.Set(api.KeyPrecursor, origMeta.Zid.String())
	return domain.Zettel{Meta: m, Content: domain.NewContent(nil)}
}

// PrepareNew the zettel for further modification.
func (*CreateZettel) PrepareNew(origZettel domain.Zettel) domain.Zettel {
	m := meta.New(id.Invalid)
	om := origZettel.Meta
	m.SetNonEmpty(api.KeyTitle, om.GetDefault(api.KeyTitle, ""))
	m.SetNonEmpty(api.KeyRole, om.GetDefault(api.KeyRole, ""))
	m.SetNonEmpty(api.KeyTags, om.GetDefault(api.KeyTags, ""))
	m.SetNonEmpty(api.KeySyntax, om.GetDefault(api.KeySyntax, ""))

	const prefixLen = len(meta.NewPrefix)
	for _, pair := range om.PairsRest() {
		if key := pair.Key; len(key) > prefixLen && key[0:prefixLen] == meta.NewPrefix {
			m.Set(key[prefixLen:], pair.Value)
		}
	}
	content := origZettel.Content
	content.TrimSpace()
	return domain.Zettel{Meta: m, Content: content}
}

func prependTitle(title, s0, s1 string) string {
	if len(title) > 0 {
		return s1 + title
	}
	return s0
}

func copyReadonly(string) string {
	// Currently, "false" is a safe value.
	//
	// If the current user and its role is known, a mor elaborative calculation
	// could be done: set it to a value, so that the current user will be able
	// to modify it later.
	return api.ValueFalse
}

// Run executes the use case.
func (uc *CreateZettel) Run(ctx context.Context, zettel domain.Zettel) (id.Zid, error) {
	m := zettel.Meta
	if m.Zid.IsValid() {
		return m.Zid, nil // TODO: new error: already exists
	}
	if title, ok := m.Get(api.KeyTitle); !ok || title == "" {
		m.SetNonEmpty(api.KeyTitle, uc.rtConfig.GetDefaultTitle())
	}
	if syntax, ok := m.Get(api.KeySyntax); !ok || syntax == "" {
		m.SetNonEmpty(api.KeySyntax, uc.rtConfig.GetDefaultSyntax())
	}

	m.Delete(api.KeyModified)
	m.YamlSep = uc.rtConfig.GetYAMLHeader()

	zettel.Content.TrimSpace()
	zid, err := uc.port.CreateZettel(ctx, zettel)
	uc.log.Info().User(ctx).Zid(zid).Err(err).Msg("Create zettel")
	return zid, err
}
