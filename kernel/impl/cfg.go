//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
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

// Predefined Metadata keys for runtime configuration
// See: https://zettelstore.de/manual/h/00001004020000
const (
	keyDefaultCopyright  = "default-copyright"
	keyDefaultLang       = "default-lang"
	keyDefaultLicense    = "default-license"
	keyDefaultRole       = "default-role"
	keyDefaultSyntax     = "default-syntax"
	keyDefaultTitle      = "default-title"
	keyDefaultVisibility = "default-visibility"
	keyExpertMode        = "expert-mode"
	keyFooterHTML        = "footer-html"
	keyHomeZettel        = "home-zettel"
	keyMarkerExternal    = "marker-external"
	keyMaxTransclusions  = "max-transclusions"
	keySiteName          = "site-name"
	keyYAMLHeader        = "yaml-header"
	keyZettelFileSyntax  = "zettel-file-syntax"
)

func (cs *configService) Initialize(logger *logger.Logger) {
	cs.logger = logger
	cs.descr = descriptionMap{
		keyDefaultCopyright: {"Default copyright", parseString, true},
		keyDefaultLang:      {"Default language", parseString, true},
		keyDefaultLicense:   {"Default license", parseString, true},
		keyDefaultRole:      {"Default role", parseString, true},
		keyDefaultSyntax:    {"Default syntax", parseString, true},
		keyDefaultTitle:     {"Default title", parseString, true},
		keyDefaultVisibility: {
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
		keyExpertMode:       {"Expert mode", parseBool, true},
		keyFooterHTML:       {"Footer HTML", parseString, true},
		keyHomeZettel:       {"Home zettel", parseZid, true},
		keyMarkerExternal:   {"Marker external URL", parseString, true},
		keyMaxTransclusions: {"Maximum transclusions", parseInt, true},
		keySiteName:         {"Site name", parseString, true},
		keyYAMLHeader:       {"YAML header", parseBool, true},
		keyZettelFileSyntax: {
			"Zettel file syntax",
			func(val string) interface{} { return strings.Fields(val) },
			true,
		},
		kernel.ConfigSimpleMode: {"Simple mode", cs.noFrozen(parseBool), true},
	}
	cs.next = interfaceMap{
		keyDefaultCopyright:     "",
		keyDefaultLang:          api.ValueLangEN,
		keyDefaultLicense:       "",
		keyDefaultRole:          api.ValueRoleZettel,
		keyDefaultSyntax:        api.ValueSyntaxZmk,
		keyDefaultTitle:         "Untitled",
		keyDefaultVisibility:    meta.VisibilityLogin,
		keyExpertMode:           false,
		keyFooterHTML:           "",
		keyHomeZettel:           id.DefaultHomeZid,
		keyMarkerExternal:       "&#10138;",
		keyMaxTransclusions:     1024,
		keySiteName:             "Zettelstore",
		keyYAMLHeader:           false,
		keyZettelFileSyntax:     nil,
		kernel.ConfigSimpleMode: false,
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
	cs.rtConfig = newConfig(cs.logger, data)
	cs.mxService.Unlock()
	return nil
}

func (cs *configService) IsStarted() bool {
	cs.mxService.RLock()
	defer cs.mxService.RUnlock()
	return cs.rtConfig != nil
}

func (cs *configService) Stop(*myKernel) {
	cs.logger.Info().Msg("Stop Service")
	cs.mxService.Lock()
	cs.rtConfig = nil
	cs.mxService.Unlock()
}

func (*configService) GetStatistics() []kernel.KeyValue {
	return nil
}

func (cs *configService) setBox(mgr box.Manager) {
	cs.rtConfig.setBox(mgr)
}

// myConfig contains all runtime configuration data relevant for the software.
type myConfig struct {
	log  *logger.Logger
	mx   sync.RWMutex
	orig *meta.Meta
	data *meta.Meta
}

// New creates a new Config value.
func newConfig(logger *logger.Logger, orig *meta.Meta) *myConfig {
	cfg := myConfig{
		log:  logger,
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
	for _, pair := range cfg.data.Pairs() {
		key := pair.Key
		if val, ok := m.Get(key); ok {
			cfg.data.Set(key, val)
		} else if defVal, defFound := cfg.orig.Get(key); defFound {
			cfg.data.Set(key, defVal)
		}
	}
	cfg.mx.Unlock()
	return nil
}

func (cfg *myConfig) observe(ci box.UpdateInfo) {
	if ci.Reason == box.OnReload || ci.Zid == id.ConfigurationZid {
		cfg.log.Debug().Uint("reason", uint64(ci.Reason)).Zid(ci.Zid).Msg("observe")
		go func() { cfg.doUpdate(ci.Box) }()
	}
}

var defaultKeys = map[string]string{
	api.KeyCopyright:  keyDefaultCopyright,
	api.KeyLang:       keyDefaultLang,
	api.KeyLicense:    keyDefaultLicense,
	api.KeyRole:       keyDefaultRole,
	api.KeySyntax:     keyDefaultSyntax,
	api.KeyTitle:      keyDefaultTitle,
	api.KeyVisibility: keyDefaultVisibility,
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
func (cfg *myConfig) GetDefaultTitle() string { return cfg.getString(keyDefaultTitle) }

// GetDefaultRole returns the current value of the "default-role" key.
func (cfg *myConfig) GetDefaultRole() string { return cfg.getString(keyDefaultRole) }

// GetDefaultSyntax returns the current value of the "default-syntax" key.
func (cfg *myConfig) GetDefaultSyntax() string { return cfg.getString(keyDefaultSyntax) }

// GetDefaultLang returns the current value of the "default-lang" key.
func (cfg *myConfig) GetDefaultLang() string { return cfg.getString(keyDefaultLang) }

// GetSiteName returns the current value of the "site-name" key.
func (cfg *myConfig) GetSiteName() string { return cfg.getString(keySiteName) }

// GetHomeZettel returns the value of the "home-zettel" key.
func (cfg *myConfig) GetHomeZettel() id.Zid {
	val := cfg.getString(keyHomeZettel)
	if homeZid, err := id.Parse(val); err == nil {
		return homeZid
	}
	cfg.mx.RLock()
	val, _ = cfg.orig.Get(keyHomeZettel)
	homeZid, _ := id.Parse(val)
	cfg.mx.RUnlock()
	return homeZid
}

// GetDefaultVisibility returns the default value for zettel visibility.
func (cfg *myConfig) GetDefaultVisibility() meta.Visibility {
	val := cfg.getString(keyDefaultVisibility)
	if vis := meta.GetVisibility(val); vis != meta.VisibilityUnknown {
		return vis
	}
	cfg.mx.RLock()
	val, _ = cfg.orig.Get(keyDefaultVisibility)
	vis := meta.GetVisibility(val)
	cfg.mx.RUnlock()
	return vis
}

// GetMaxTransclusions return the maximum number of indirect transclusions.
func (cfg *myConfig) GetMaxTransclusions() int {
	cfg.mx.RLock()
	val, ok := cfg.data.GetNumber(keyMaxTransclusions)
	cfg.mx.RUnlock()
	if ok && val > 0 {
		return val
	}
	return 1024
}

// GetYAMLHeader returns the current value of the "yaml-header" key.
func (cfg *myConfig) GetYAMLHeader() bool { return cfg.getBool(keyYAMLHeader) }

// GetMarkerExternal returns the current value of the "marker-external" key.
func (cfg *myConfig) GetMarkerExternal() string {
	return cfg.getString(keyMarkerExternal)
}

// GetFooterHTML returns HTML code that should be embedded into the footer
// of each WebUI page.
func (cfg *myConfig) GetFooterHTML() string { return cfg.getString(keyFooterHTML) }

// GetZettelFileSyntax returns the current value of the "zettel-file-syntax" key.
func (cfg *myConfig) GetZettelFileSyntax() []string {
	cfg.mx.RLock()
	defer cfg.mx.RUnlock()
	return cfg.data.GetListOrNil(keyZettelFileSyntax)
}

// --- AuthConfig

// GetSimpleMode returns true if system tuns in simple-mode.
func (cfg *myConfig) GetSimpleMode() bool { return cfg.getBool(kernel.ConfigSimpleMode) }

// GetExpertMode returns the current value of the "expert-mode" key.
func (cfg *myConfig) GetExpertMode() bool { return cfg.getBool(keyExpertMode) }

// GetVisibility returns the visibility value, or "login" if none is given.
func (cfg *myConfig) GetVisibility(m *meta.Meta) meta.Visibility {
	if val, ok := m.Get(api.KeyVisibility); ok {
		if vis := meta.GetVisibility(val); vis != meta.VisibilityUnknown {
			return vis
		}
	}
	return cfg.GetDefaultVisibility()
}
