//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package impl provides the main internal service implementation.
package impl

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/change"
	"zettelstore.de/z/service"
)

type cfgSub struct {
	subConfig
	mxService sync.RWMutex
	rtConfig  *myConfig
}

func (cs *cfgSub) Initialize() {
	cs.descr = descriptionMap{
		meta.KeyCopyright:  {"Copyright", parseString, true},
		meta.KeyExpertMode: {"Expert mode", parseBool, true},
		meta.KeyFooterHTML: {"Footer HTML", parseString, true},
		meta.KeyHomeZettel: {"Home zettel", parseZid, true},
		meta.KeyListPageSize: {
			"List page size",
			func(val string) interface{} {
				iVal, err := strconv.Atoi(val)
				if err != nil {
					return nil
				}
				return iVal
			},
			true,
		},
		meta.KeyDefaultLang:    {"Language", parseString, true},
		meta.KeyMarkerExternal: {"Marker external URL", parseString, true},
		meta.KeyDefaultRole:    {"Default role", parseString, true},
		meta.KeySiteName:       {"Site name", parseString, true},
		meta.KeyDefaultSyntax:  {"Default syntax", parseString, true},
		meta.KeyDefaultTitle:   {"Default title", parseString, true},
		meta.KeyDefaultVisibility: {
			"Default zettel visibility",
			func(val string) interface{} {
				vis := meta.GetVisibility(val)
				if vis == meta.VisibilityUnknown {
					return nil
				}
				return vis
			},
			true,
		},
		meta.KeyYAMLHeader: {"YAML header", parseBool, true},
		meta.KeyZettelFileSyntax: {
			"Zettel file syntax",
			func(val string) interface{} { return strings.Fields(val) },
			true,
		},
	}
	cs.next = interfaceMap{
		meta.KeyCopyright:         "",
		meta.KeyExpertMode:        false,
		meta.KeyFooterHTML:        "",
		meta.KeyHomeZettel:        id.DefaultHomeZid,
		meta.KeyListPageSize:      0,
		meta.KeyDefaultLang:       meta.ValueLangEN,
		meta.KeyMarkerExternal:    "&#10138;",
		meta.KeyDefaultRole:       meta.ValueRoleZettel,
		meta.KeySiteName:          "Zettelstore",
		meta.KeyDefaultSyntax:     meta.ValueSyntaxZmk,
		meta.KeyDefaultTitle:      "Untitled",
		meta.KeyDefaultVisibility: meta.VisibilityLogin,
		meta.KeyYAMLHeader:        false,
		meta.KeyZettelFileSyntax:  nil,
	}
}

func (cs *cfgSub) Start(srv *myService) error {
	srv.doLog("Start Config Service")
	data := meta.New(id.ConfigurationZid)
	for _, kv := range cs.GetNextConfigList() {
		data.Set(kv.Key, fmt.Sprintf("%v", kv.Value))
	}
	cs.mxService.Lock()
	cs.rtConfig = newConfig(data)
	cs.mxService.Unlock()
	return nil
}

func (cs *cfgSub) IsStarted() bool {
	cs.mxService.RLock()
	defer cs.mxService.RUnlock()
	return cs.rtConfig != nil
}

func (cs *cfgSub) Stop(srv *myService) error {
	srv.doLog("Stop Config Service")
	cs.mxService.Lock()
	cs.rtConfig = nil
	cs.mxService.Unlock()
	return nil
}

func (cs *cfgSub) GetStatistics() []service.KeyValue {
	return nil
}

func (cs *cfgSub) setPlace(mgr place.Manager) {
	cs.rtConfig.setPlace(mgr)
}

// myConfig contains all runtime configuration data relevant for the software.
type myConfig struct {
	place place.Place
	orig  *meta.Meta
	data  *meta.Meta
	mx    sync.RWMutex
}

// New creates a new Config value.
func newConfig(orig *meta.Meta) *myConfig {
	cfg := myConfig{
		orig: orig,
		data: orig.Clone(),
	}
	return &cfg
}
func (cfg *myConfig) setPlace(mgr place.Manager) {
	cfg.mx.Lock()
	cfg.place = mgr
	cfg.mx.Unlock()
	mgr.RegisterObserver(cfg.observe)
	cfg.doUpdate()
}

func (cfg *myConfig) doUpdate() error {
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

func (cfg *myConfig) observe(ci change.Info) {
	if ci.Reason == change.OnReload || ci.Zid == id.ConfigurationZid {
		go func() { cfg.doUpdate() }()
	}
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
func (cfg *myConfig) AddDefaultValues(m *meta.Meta) *meta.Meta {
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

func (cfg *myConfig) getString(key string) string {
	cfg.mx.RLock()
	val, _ := cfg.data.Get(key)
	cfg.mx.RUnlock()
	return val
}
func (cfg *myConfig) getBool(key string) bool {
	cfg.mx.RLock()
	val := cfg.data.GetBool(key)
	cfg.mx.RUnlock()
	return val
}

// GetDefaultTitle returns the current value of the "default-title" key.
func (cfg *myConfig) GetDefaultTitle() string { return cfg.getString(meta.KeyDefaultTitle) }

// GetDefaultRole returns the current value of the "default-role" key.
func (cfg *myConfig) GetDefaultRole() string { return cfg.getString(meta.KeyDefaultRole) }

// GetDefaultSyntax returns the current value of the "default-syntax" key.
func (cfg *myConfig) GetDefaultSyntax() string { return cfg.getString(meta.KeyDefaultSyntax) }

// GetDefaultLang returns the current value of the "default-lang" key.
func (cfg *myConfig) GetDefaultLang() string { return cfg.getString(meta.KeyDefaultLang) }

// GetExpertMode returns the current value of the "expert-mode" key
func (cfg *myConfig) GetExpertMode() bool { return cfg.getBool(meta.KeyExpertMode) }

// GetSiteName returns the current value of the "site-name" key.
func (cfg *myConfig) GetSiteName() string { return cfg.getString(meta.KeySiteName) }

// GetHomeZettel returns the value of the "home-zettel" key.
func (cfg *myConfig) GetHomeZettel() id.Zid {
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

// GetDefaultVisibility returns the default value for zettel visibility.
func (cfg *myConfig) GetDefaultVisibility() meta.Visibility {
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
func (cfg *myConfig) GetYAMLHeader() bool { return cfg.getBool(meta.KeyYAMLHeader) }

// GetMarkerExternal returns the current value of the "marker-external" key.
func (cfg *myConfig) GetMarkerExternal() string {
	return cfg.getString(meta.KeyMarkerExternal)
}

// GetFooterHTML returns HTML code that should be embedded into the footer
// of each WebUI page.
func (cfg *myConfig) GetFooterHTML() string { return cfg.getString(meta.KeyFooterHTML) }

// GetListPageSize returns the maximum length of a list to be returned in WebUI.
// A value less or equal to zero signals no limit.
func (cfg *myConfig) GetListPageSize() int {
	cfg.mx.RLock()
	defer cfg.mx.RUnlock()

	if value, ok := cfg.data.GetNumber(meta.KeyListPageSize); ok {
		return value
	}
	value, _ := cfg.orig.GetNumber(meta.KeyListPageSize)
	return value
}

// GetZettelFileSyntax returns the current value of the "zettel-file-syntax" key.
func (cfg *myConfig) GetZettelFileSyntax() []string {
	cfg.mx.RLock()
	defer cfg.mx.RUnlock()
	return cfg.data.GetListOrNil(meta.KeyZettelFileSyntax)
}
