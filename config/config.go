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
	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

// Config allows to retrieve all defined configuration values that can be changed during runtime.
type Config interface {
	AuthConfig

	// AddDefaultValues enriches the given meta data with its default values.
	AddDefaultValues(m *meta.Meta) *meta.Meta

	// GetDefaultLang returns the current value of the "default-lang" key.
	GetDefaultLang() string

	// GetSiteName returns the current value of the "site-name" key.
	GetSiteName() string

	// GetHomeZettel returns the value of the "home-zettel" key.
	GetHomeZettel() id.Zid

	// GetDefaultVisibility returns the default value for zettel visibility.
	GetDefaultVisibility() meta.Visibility

	// GetMaxTransclusions return the maximum number of indirect transclusions.
	GetMaxTransclusions() int

	// GetYAMLHeader returns the current value of the "yaml-header" key.
	GetYAMLHeader() bool

	// GetZettelFileSyntax returns the current value of the "zettel-file-syntax" key.
	GetZettelFileSyntax() []string

	// GetMarkerExternal returns the current value of the "marker-external" key.
	GetMarkerExternal() string

	// GetFooterHTML returns HTML code that should be embedded into the footer
	// of each WebUI page.
	GetFooterHTML() string
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
// no such value, GetDefaultLang is returned.
func GetLang(m *meta.Meta, cfg Config) string {
	if val, ok := m.Get(api.KeyLang); ok {
		return val
	}
	return cfg.GetDefaultLang()
}
