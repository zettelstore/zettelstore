//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package impl

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/logger"
)

type configService struct {
	srvConfig
	mxService sync.RWMutex
	rtConfig  *myConfig
}

func (cs *configService) Initialize(logger *logger.Logger) {
	cs.logger = logger
	cs.descr = descriptionMap{
		api.KeyDefaultCopyright: {"Default copyright", parseString, true},
		api.KeyDefaultLang:      {"Default language", parseString, true},
		api.KeyDefaultRole:      {"Default role", parseString, true},
		api.KeyDefaultSyntax:    {"Default syntax", parseString, true},
		api.KeyDefaultTitle:     {"Default title", parseString, true},
		api.KeyDefaultVisibility: {
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
		api.KeyExpertMode:       {"Expert mode", parseBool, true},
		api.KeyFooterHTML:       {"Footer HTML", parseString, true},
		api.KeyHomeZettel:       {"Home zettel", parseZid, true},
		api.KeyMarkerExternal:   {"Marker external URL", parseString, true},
		api.KeyMaxTransclusions: {"Maximum transclusions", parseInt, true},
		api.KeySiteName:         {"Site name", parseString, true},
		api.KeyYAMLHeader:       {"YAML header", parseBool, true},
		api.KeyZettelFileSyntax: {
			"Zettel file syntax",
			func(val string) interface{} { return strings.Fields(val) },
			true,
		},
	}
	cs.next = interfaceMap{
		api.KeyDefaultCopyright:  "",
		api.KeyDefaultLang:       api.ValueLangEN,
		api.KeyDefaultRole:       api.ValueRoleZettel,
		api.KeyDefaultSyntax:     api.ValueSyntaxZmk,
		api.KeyDefaultTitle:      "Untitled",
		api.KeyDefaultVisibility: meta.VisibilityLogin,
		api.KeyExpertMode:        false,
		api.KeyFooterHTML:        "",
		api.KeyHomeZettel:        id.DefaultHomeZid,
		api.KeyMarkerExternal:    "&#10138;",
		api.KeyMaxTransclusions:  1024,
		api.KeySiteName:          "Zettelstore",
		api.KeyYAMLHeader:        false,
		api.KeyZettelFileSyntax:  nil,
	}
}
func (cs *configService) GetLogger() *logger.Logger { return cs.logger }

func (cs *configService) Start(*myKernel) error {
	cs.logger.Info().Msg("Start Service")
	data := meta.New(id.ConfigurationZid)
	for _, kv := range cs.GetNextConfigList() {
		data.Set(kv.Key, fmt.Sprintf("%v", kv.Value))
	}
	cs.mxService.Lock()
	cs.rtConfig = newConfig(data)
	cs.mxService.Unlock()
	return nil
}

func (cs *configService) IsStarted() bool {
	cs.mxService.RLock()
	defer cs.mxService.RUnlock()
	return cs.rtConfig != nil
}

func (cs *configService) Stop(*myKernel) error {
	cs.logger.Info().Msg("Stop Service")
	cs.mxService.Lock()
	cs.rtConfig = nil
	cs.mxService.Unlock()
	return nil
}

func (*configService) GetStatistics() []kernel.KeyValue {
	return nil
}

func (cs *configService) setBox(mgr box.Manager) {
	cs.rtConfig.setBox(mgr)
}

// myConfig contains all runtime configuration data relevant for the software.
type myConfig struct {
	mx   sync.RWMutex
	orig *meta.Meta
	data *meta.Meta
}

// New creates a new Config value.
func newConfig(orig *meta.Meta) *myConfig {
	cfg := myConfig{
		orig: orig,
		data: orig.Clone(),
	}
	return &cfg
}
func (cfg *myConfig) setBox(mgr box.Manager) {
	mgr.RegisterObserver(cfg.observe)
	cfg.doUpdate(mgr)
}

func (cfg *myConfig) doUpdate(p box.Box) error {
	m, err := p.GetMeta(context.Background(), cfg.data.Zid)
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

func (cfg *myConfig) observe(ci box.UpdateInfo) {
	if ci.Reason == box.OnReload || ci.Zid == id.ConfigurationZid {
		go func() { cfg.doUpdate(ci.Box) }()
	}
}

var defaultKeys = map[string]string{
	api.KeyCopyright: api.KeyDefaultCopyright,
	api.KeyLang:      api.KeyDefaultLang,
	api.KeyLicense:   api.KeyDefaultLicense,
	api.KeyRole:      api.KeyDefaultRole,
	api.KeySyntax:    api.KeyDefaultSyntax,
	api.KeyTitle:     api.KeyDefaultTitle,
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
			if val, ok2 := cfg.data.Get(d); ok2 && val != "" {
				if result == m {
					result = m.Clone()
				}
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
func (cfg *myConfig) GetDefaultTitle() string { return cfg.getString(api.KeyDefaultTitle) }

// GetDefaultRole returns the current value of the "default-role" key.
func (cfg *myConfig) GetDefaultRole() string { return cfg.getString(api.KeyDefaultRole) }

// GetDefaultSyntax returns the current value of the "default-syntax" key.
func (cfg *myConfig) GetDefaultSyntax() string { return cfg.getString(api.KeyDefaultSyntax) }

// GetDefaultLang returns the current value of the "default-lang" key.
func (cfg *myConfig) GetDefaultLang() string { return cfg.getString(api.KeyDefaultLang) }

// GetSiteName returns the current value of the "site-name" key.
func (cfg *myConfig) GetSiteName() string { return cfg.getString(api.KeySiteName) }

// GetHomeZettel returns the value of the "home-zettel" key.
func (cfg *myConfig) GetHomeZettel() id.Zid {
	val := cfg.getString(api.KeyHomeZettel)
	if homeZid, err := id.Parse(val); err == nil {
		return homeZid
	}
	cfg.mx.RLock()
	val, _ = cfg.orig.Get(api.KeyHomeZettel)
	homeZid, _ := id.Parse(val)
	cfg.mx.RUnlock()
	return homeZid
}

// GetDefaultVisibility returns the default value for zettel visibility.
func (cfg *myConfig) GetDefaultVisibility() meta.Visibility {
	val := cfg.getString(api.KeyDefaultVisibility)
	if vis := meta.GetVisibility(val); vis != meta.VisibilityUnknown {
		return vis
	}
	cfg.mx.RLock()
	val, _ = cfg.orig.Get(api.KeyDefaultVisibility)
	vis := meta.GetVisibility(val)
	cfg.mx.RUnlock()
	return vis
}

// GetMaxTransclusions return the maximum number of indirect transclusions.
func (cfg *myConfig) GetMaxTransclusions() int {
	cfg.mx.RLock()
	val, ok := cfg.data.GetNumber(api.KeyMaxTransclusions)
	cfg.mx.RUnlock()
	if ok && val > 0 {
		return val
	}
	return 1024
}

// GetYAMLHeader returns the current value of the "yaml-header" key.
func (cfg *myConfig) GetYAMLHeader() bool { return cfg.getBool(api.KeyYAMLHeader) }

// GetMarkerExternal returns the current value of the "marker-external" key.
func (cfg *myConfig) GetMarkerExternal() string {
	return cfg.getString(api.KeyMarkerExternal)
}

// GetFooterHTML returns HTML code that should be embedded into the footer
// of each WebUI page.
func (cfg *myConfig) GetFooterHTML() string { return cfg.getString(api.KeyFooterHTML) }

// GetZettelFileSyntax returns the current value of the "zettel-file-syntax" key.
func (cfg *myConfig) GetZettelFileSyntax() []string {
	cfg.mx.RLock()
	defer cfg.mx.RUnlock()
	return cfg.data.GetListOrNil(api.KeyZettelFileSyntax)
}

// --- AuthConfig

// GetExpertMode returns the current value of the "expert-mode" key
func (cfg *myConfig) GetExpertMode() bool { return cfg.getBool(api.KeyExpertMode) }

// GetVisibility returns the visibility value, or "login" if none is given.
func (cfg *myConfig) GetVisibility(m *meta.Meta) meta.Visibility {
	if val, ok := m.Get(api.KeyVisibility); ok {
		if vis := meta.GetVisibility(val); vis != meta.VisibilityUnknown {
			return vis
		}
	}
	return cfg.GetDefaultVisibility()
}
