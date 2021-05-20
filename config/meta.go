//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
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
	"zettelstore.de/z/domain/meta"
)

var mapDefaultKeys = map[string]func() string{
	meta.KeyCopyright: GetDefaultCopyright,
	meta.KeyLang:      GetDefaultLang,
	meta.KeyLicense:   GetDefaultLicense,
	meta.KeyRole:      GetDefaultRole,
	meta.KeySyntax:    GetDefaultSyntax,
	meta.KeyTitle:     GetDefaultTitle,
}

// AddDefaultValues enriches the given meta data with its default values.
func AddDefaultValues(m *meta.Meta) *meta.Meta {
	result := m
	for k, f := range mapDefaultKeys {
		if _, ok := result.Get(k); !ok {
			if result == m {
				result = m.Clone()
			}
			if val := f(); len(val) > 0 || m.Type(k) == meta.TypeEmpty {
				result.Set(k, val)
			}
		}
	}
	return result
}

// GetTitle returns the value of the "title" key of the given meta. If there
// is no such value, GetDefaultTitle is returned.
func GetTitle(m *meta.Meta) string {
	if syntax, ok := m.Get(meta.KeyTitle); ok && len(syntax) > 0 {
		return syntax
	}
	return GetDefaultTitle()
}

// GetRole returns the value of the "role" key of the given meta. If there
// is no such value, GetDefaultRole is returned.
func GetRole(m *meta.Meta) string {
	if syntax, ok := m.Get(meta.KeyRole); ok && len(syntax) > 0 {
		return syntax
	}
	return GetDefaultRole()
}

// GetSyntax returns the value of the "syntax" key of the given meta. If there
// is no such value, GetDefaultSyntax is returned.
func GetSyntax(m *meta.Meta) string {
	if syntax, ok := m.Get(meta.KeySyntax); ok && len(syntax) > 0 {
		return syntax
	}
	return GetDefaultSyntax()
}

// GetLang returns the value of the "lang" key of the given meta. If there is
// no such value, GetDefaultLang is returned.
func GetLang(m *meta.Meta) string {
	if lang, ok := m.Get(meta.KeyLang); ok && len(lang) > 0 {
		return lang
	}
	return GetDefaultLang()
}

// GetVisibility returns the visibility value, or "login" if none is given.
func GetVisibility(m *meta.Meta) meta.Visibility {
	if val, ok := m.Get(meta.KeyVisibility); ok {
		if vis := meta.GetVisibility(val); vis != meta.VisibilityUnknown {
			return vis
		}
	}
	return GetDefaultVisibility()
}
