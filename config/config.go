//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package config provides functions to retrieve runtime configuration data.
package config

import (
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

// Config allows to retrieve all defined configuration values that can be changed during runtime.
type Config interface {
	// AddDefaultValues enriches the given meta data with its default values.
	AddDefaultValues(m *meta.Meta) *meta.Meta

	// GetDefaultTitle returns the current value of the "default-title" key.
	GetDefaultTitle() string

	// GetDefaultRole returns the current value of the "default-role" key.
	GetDefaultRole() string

	// GetDefaultSyntax returns the current value of the "default-syntax" key.
	GetDefaultSyntax() string

	// GetDefaultLang returns the current value of the "default-lang" key.
	GetDefaultLang() string

	// GetExpertMode returns the current value of the "expert-mode" key
	GetExpertMode() bool

	// GetSiteName returns the current value of the "site-name" key.
	GetSiteName() string

	// GetHomeZettel returns the value of the "home-zettel" key.
	GetHomeZettel() id.Zid

	// GetDefaultVisibility returns the default value for zettel visibility.
	GetDefaultVisibility() meta.Visibility

	// GetYAMLHeader returns the current value of the "yaml-header" key.
	GetYAMLHeader() bool

	// GetZettelFileSyntax returns the current value of the "zettel-file-syntax" key.
	GetZettelFileSyntax() []string

	// GetMarkerExternal returns the current value of the "marker-external" key.
	GetMarkerExternal() string

	// GetFooterHTML returns HTML code that should be embedded into the footer
	// of each WebUI page.
	GetFooterHTML() string

	// GetListPageSize returns the maximum length of a list to be returned in WebUI.
	// A value less or equal to zero signals no limit.
	GetListPageSize() int
}

// GetTitle returns the value of the "title" key of the given meta. If there
// is no such value, GetDefaultTitle is returned.
func GetTitle(m *meta.Meta, cfg Config) string {
	if val, ok := m.Get(meta.KeyTitle); ok {
		return val
	}
	return cfg.GetDefaultTitle()
}

// GetRole returns the value of the "role" key of the given meta. If there
// is no such value, GetDefaultRole is returned.
func GetRole(m *meta.Meta, cfg Config) string {
	if val, ok := m.Get(meta.KeyRole); ok {
		return val
	}
	return cfg.GetDefaultRole()
}

// GetSyntax returns the value of the "syntax" key of the given meta. If there
// is no such value, GetDefaultSyntax is returned.
func GetSyntax(m *meta.Meta, cfg Config) string {
	if val, ok := m.Get(meta.KeySyntax); ok {
		return val
	}
	return cfg.GetDefaultSyntax()
}

// GetLang returns the value of the "lang" key of the given meta. If there is
// no such value, GetDefaultLang is returned.
func GetLang(m *meta.Meta, cfg Config) string {
	if val, ok := m.Get(meta.KeyLang); ok {
		return val
	}
	return cfg.GetDefaultLang()
}

// GetVisibility returns the visibility value, or "login" if none is given.
func GetVisibility(m *meta.Meta, cfg Config) meta.Visibility {
	if val, ok := m.Get(meta.KeyVisibility); ok {
		if vis := meta.GetVisibility(val); vis != meta.VisibilityUnknown {
			return vis
		}
	}
	return cfg.GetDefaultVisibility()
}
