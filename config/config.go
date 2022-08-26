//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package config provides functions to retrieve runtime configuration data.
package config

import (
	"context"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

// Key values that are supported by Config.Get
const (
	KeyDefaultLang    = "default-lang"
	KeyFooterHTML     = "footer-html"
	KeyMarkerExternal = "marker-external"
)

// Config allows to retrieve all defined configuration values that can be changed during runtime.
type Config interface {
	AuthConfig

	// Get returns the value of the given key. It searches first in the given metadata,
	// then in the data of the current user, and at last in the system-wide data.
	Get(ctx context.Context, m *meta.Meta, key string) string

	// AddDefaultValues enriches the given meta data with its default values.
	AddDefaultValues(m *meta.Meta) *meta.Meta

	// GetSiteName returns the current value of the "site-name" key.
	GetSiteName() string

	// GetHomeZettel returns the value of the "home-zettel" key.
	GetHomeZettel() id.Zid

	// GetMaxTransclusions return the maximum number of indirect transclusions.
	GetMaxTransclusions() int

	// GetYAMLHeader returns the current value of the "yaml-header" key.
	GetYAMLHeader() bool

	// GetZettelFileSyntax returns the current value of the "zettel-file-syntax" key.
	GetZettelFileSyntax() []string
}

// AuthConfig are relevant configuration values for authentication.
type AuthConfig interface {
	// GetSimpleMode returns true if system tuns in simple-mode.
	GetSimpleMode() bool

	// GetExpertMode returns the current value of the "expert-mode" key.
	GetExpertMode() bool

	// GetVisibility returns the visibility value of the metadata.
	GetVisibility(m *meta.Meta) meta.Visibility
}

// GetLang returns the value of the "lang" key of the given meta. If there is
// no such value, GetDefaultLang() is returned.
func GetLang(ctx context.Context, m *meta.Meta, cfg Config) string {
	if val, ok := m.Get(api.KeyLang); ok {
		return val
	}
	return GetDefaultLang(ctx, cfg)
}

// GetDefaultLang returns the configured value for a default language.
func GetDefaultLang(ctx context.Context, cfg Config) string { return cfg.Get(ctx, nil, KeyDefaultLang) }
