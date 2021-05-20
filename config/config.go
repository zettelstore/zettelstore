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
	"context"
	"sync"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/change"
)

// Config contains all runtime configuration data relevant for the software.
type Config struct {
	place place.Place
	orig  *meta.Meta
	data  *meta.Meta
	mx    sync.RWMutex
}

// New creates a new Config value.
func New(orig *meta.Meta, manager place.Manager) (*Config, error) {
	cfg := Config{
		place: manager,
		orig:  orig,
		data:  orig.Clone(),
	}
	if err := cfg.doUpdate(); err != nil {
		return nil, err
	}
	manager.RegisterObserver(cfg.observe)
	return &cfg, nil
}

func (cfg *Config) doUpdate() error {
	m, err := cfg.place.GetMeta(context.Background(), cfg.data.Zid)
	if err != nil {
		return err
	}
	cfg.mx.Lock()
	for _, pair := range cfg.data.Pairs(false) {
		if val, ok := m.Get(pair.Key); ok {
			cfg.data.Set(pair.Key, val)
		}
	}
	cfg.mx.Unlock()
	return nil
}

func (cfg *Config) observe(ci change.Info) {
	if ci.Reason == change.OnReload || ci.Zid == id.ConfigurationZid {
		go func() { cfg.doUpdate() }()
	}
}

func (cfg *Config) getString(key string) string {
	cfg.mx.RLock()
	val, _ := cfg.data.Get(key)
	cfg.mx.RUnlock()
	return val
}
func (cfg *Config) getBool(key string) bool {
	cfg.mx.RLock()
	val := cfg.data.GetBool(key)
	cfg.mx.RUnlock()
	return val
}

var defaultKeys = map[string]string{
	meta.KeyCopyright: meta.KeyDefaultCopyright,
	meta.KeyLang:      meta.KeyDefaultLang,
	meta.KeyLicense:   meta.KeyDefaultLicense,
	meta.KeyRole:      meta.KeyDefaultRole,
	meta.KeySyntax:    meta.KeyDefaultSyntax,
	meta.KeyTitle:     meta.KeyDefaultTitle,
}

// AddDefaultValues enriches the given meta data with its default values.
func (cfg *Config) AddDefaultValues(m *meta.Meta) *meta.Meta {
	if cfg == nil {
		return m
	}
	result := m
	cfg.mx.RLock()
	for k, d := range defaultKeys {
		if _, ok := result.Get(k); !ok {
			if result == m {
				result = m.Clone()
			}
			if val, ok := cfg.data.Get(d); ok {
				result.Set(k, val)
			}
		}
	}
	cfg.mx.RUnlock()
	return result
}

// GetTitle returns the value of the "title" key of the given meta. If there
// is no such value, GetDefaultTitle is returned.
func (cfg *Config) GetTitle(m *meta.Meta) string {
	if val, ok := m.Get(meta.KeyTitle); ok {
		return val
	}
	return cfg.GetDefaultTitle()
}

// GetDefaultTitle returns the current value of the "default-title" key.
func (cfg *Config) GetDefaultTitle() string { return cfg.getString(meta.KeyDefaultTitle) }

// GetRole returns the value of the "role" key of the given meta. If there
// is no such value, GetDefaultRole is returned.
func (cfg *Config) GetRole(m *meta.Meta) string {
	if val, ok := m.Get(meta.KeyRole); ok {
		return val
	}
	return cfg.GetDefaultRole()
}

// GetDefaultRole returns the current value of the "default-role" key.
func (cfg *Config) GetDefaultRole() string { return cfg.getString(meta.KeyDefaultRole) }

// GetSyntax returns the value of the "syntax" key of the given meta. If there
// is no such value, GetDefaultSyntax is returned.
func (cfg *Config) GetSyntax(m *meta.Meta) string {
	if val, ok := m.Get(meta.KeySyntax); ok {
		return val
	}
	return cfg.GetDefaultSyntax()
}

// GetDefaultSyntax returns the current value of the "default-syntax" key.
func (cfg *Config) GetDefaultSyntax() string { return cfg.getString(meta.KeyDefaultSyntax) }

// GetLang returns the value of the "lang" key of the given meta. If there is
// no such value, GetDefaultLang is returned.
func (cfg *Config) GetLang(m *meta.Meta) string {
	if val, ok := m.Get(meta.KeyLang); ok {
		return val
	}
	return cfg.GetDefaultLang()
}

// GetDefaultLang returns the current value of the "default-lang" key.
func (cfg *Config) GetDefaultLang() string { return cfg.getString(meta.KeyDefaultLang) }

// GetExpertMode returns the current value of the "expert-mode" key
func (cfg *Config) GetExpertMode() bool { return cfg.getBool(meta.KeyExpertMode) }

// GetSiteName returns the current value of the "site-name" key.
func (cfg *Config) GetSiteName() string { return cfg.getString(meta.KeySiteName) }

// GetHomeZettel returns the value of the "home-zettel" key.
func (cfg *Config) GetHomeZettel() id.Zid {
	val := cfg.getString(meta.KeyHomeZettel)
	if homeZid, err := id.Parse(val); err == nil {
		return homeZid
	}
	cfg.mx.RLock()
	val, _ = cfg.orig.Get(meta.KeyHomeZettel)
	homeZid, _ := id.Parse(val)
	cfg.mx.RUnlock()
	return homeZid
}

// GetVisibility returns the visibility value, or "login" if none is given.
func (cfg *Config) GetVisibility(m *meta.Meta) meta.Visibility {
	if val, ok := m.Get(meta.KeyVisibility); ok {
		if vis := meta.GetVisibility(val); vis != meta.VisibilityUnknown {
			return vis
		}
	}
	return cfg.GetDefaultVisibility()
}

// GetDefaultVisibility returns the default value for zettel visibility.
func (cfg *Config) GetDefaultVisibility() meta.Visibility {
	val := cfg.getString(meta.KeyDefaultVisibility)
	if vis := meta.GetVisibility(val); vis != meta.VisibilityUnknown {
		return vis
	}
	cfg.mx.RLock()
	val, _ = cfg.orig.Get(meta.KeyDefaultVisibility)
	vis := meta.GetVisibility(val)
	cfg.mx.RUnlock()
	return vis
}

// GetYAMLHeader returns the current value of the "yaml-header" key.
func (cfg *Config) GetYAMLHeader() bool { return cfg.getBool(meta.KeyYAMLHeader) }

// GetMarkerExternal returns the current value of the "marker-external" key.
func (cfg *Config) GetMarkerExternal() string {
	return cfg.getString(meta.KeyMarkerExternal)
}

// GetFooterHTML returns HTML code that should be embedded into the footer
// of each WebUI page.
func (cfg *Config) GetFooterHTML() string { return cfg.getString(meta.KeyFooterHTML) }

// GetListPageSize returns the maximum length of a list to be returned in WebUI.
// A value less or equal to zero signals no limit.
func (cfg *Config) GetListPageSize() int {
	mxConfig.RLock()
	defer mxConfig.RUnlock()

	if value, ok := cfg.data.GetNumber(meta.KeyListPageSize); ok {
		return value
	}
	value, _ := cfg.orig.GetNumber(meta.KeyListPageSize)
	return value
}

// --- Configuration zettel --------------------------------------------------

var (
	mxConfig    sync.RWMutex
	configPlace place.Manager
	configMeta  *meta.Meta
)

// SetupConfiguration enables the configuration data.
func SetupConfiguration(mgr place.Manager) {
	mxConfig.Lock()
	defer mxConfig.Unlock()
	m, err := mgr.GetMeta(context.Background(), id.ConfigurationZid)
	if err != nil {
		panic(err)
	}
	configPlace = mgr
	configMeta = m
	mgr.RegisterObserver(observe)
}

// observe tracks all changes the place signals.
func observe(ci change.Info) {
	if ci.Reason == change.OnReload || ci.Zid == id.ConfigurationZid {
		go func() {
			mxConfig.Lock()
			defer mxConfig.Unlock()
			if m, err := configPlace.GetMeta(context.Background(), id.ConfigurationZid); err == nil {
				configMeta = m
			}
		}()
	}
}

// GetDefaultRole returns the current value of the "default-role" key.
func GetDefaultRole() string {
	mxConfig.RLock()
	defer mxConfig.RUnlock()
	if configMeta != nil {
		if role, ok := configMeta.Get(meta.KeyDefaultRole); ok {
			return role
		}
	}
	return meta.ValueRoleZettel
}

// GetDefaultSyntax returns the current value of the "default-syntax" key.
func GetDefaultSyntax() string {
	mxConfig.RLock()
	defer mxConfig.RUnlock()
	if configMeta != nil {
		if syntax, ok := configMeta.Get(meta.KeyDefaultSyntax); ok {
			return syntax
		}
	}
	return meta.ValueSyntaxZmk
}

// GetZettelFileSyntax returns the current value of the "zettel-file-syntax" key.
func GetZettelFileSyntax() []string {
	mxConfig.RLock()
	defer mxConfig.RUnlock()
	if configMeta != nil {
		return configMeta.GetListOrNil(meta.KeyZettelFileSyntax)
	}
	return nil
}
