//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package impl

import (
	"context"
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
	orig      *meta.Meta
}

// Predefined Metadata keys for runtime configuration
// See: https://zettelstore.de/manual/h/00001004020000
const (
	keyDefaultCopyright  = "default-copyright"
	keyDefaultLang       = "default-lang"
	keyDefaultLicense    = "default-license"
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
		keyMaxTransclusions: {"Maximum transclusions", parseInt64, true},
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
		keyDefaultVisibility:    meta.VisibilityLogin,
		keyExpertMode:           false,
		keyFooterHTML:           "",
		keyHomeZettel:           id.DefaultHomeZid,
		keyMarkerExternal:       "&#10138;",
		keyMaxTransclusions:     int64(1024),
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
		data.Set(kv.Key, kv.Value)
	}
	cs.mxService.Lock()
	cs.orig = data
	cs.mxService.Unlock()
	return nil
}

func (cs *configService) IsStarted() bool {
	cs.mxService.RLock()
	defer cs.mxService.RUnlock()
	return cs.orig != nil
}

func (cs *configService) Stop(*myKernel) {
	cs.logger.Info().Msg("Stop Service")
	cs.mxService.Lock()
	cs.orig = nil
	cs.mxService.Unlock()
}

func (*configService) GetStatistics() []kernel.KeyValue {
	return nil
}

func (cs *configService) setBox(mgr box.Manager) {
	mgr.RegisterObserver(cs.observe)
	cs.doUpdate(mgr)
}

func (cs *configService) doUpdate(p box.Box) error {
	m, err := p.GetMeta(context.Background(), cs.orig.Zid)
	if err != nil {
		return err
	}
	cs.mxService.Lock()
	for _, pair := range cs.orig.Pairs() {
		key := pair.Key
		if val, ok := m.Get(key); ok {
			cs.SetConfig(key, val)
		} else if defVal, defFound := cs.orig.Get(key); defFound {
			cs.SetConfig(key, defVal)
		}
	}
	cs.mxService.Unlock()
	cs.SwitchNextToCur() // Poor man's restart
	return nil
}

func (cs *configService) observe(ci box.UpdateInfo) {
	if ci.Reason == box.OnReload || ci.Zid == id.ConfigurationZid {
		cs.logger.Debug().Uint("reason", uint64(ci.Reason)).Zid(ci.Zid).Msg("observe")
		go func() { cs.doUpdate(ci.Box) }()
	}
}

// --- config.Config

// AddDefaultValues enriches the given meta data with its default values.
func (cs *configService) AddDefaultValues(m *meta.Meta) *meta.Meta {
	if cs == nil {
		return m
	}
	result := m
	cs.mxService.RLock()
	if _, found := m.Get(api.KeyCopyright); !found {
		result = updateMeta(m, result, api.KeyCopyright, cs.GetConfig(keyDefaultCopyright).(string))
	}
	if _, found := m.Get(api.KeyLang); !found {
		result = updateMeta(m, result, api.KeyLang, cs.GetConfig(keyDefaultLang).(string))
	}
	if _, found := m.Get(api.KeyLicense); !found {
		result = updateMeta(m, result, api.KeyLicense, cs.GetConfig(keyDefaultLicense).(string))
	}
	if _, found := m.Get(api.KeyVisibility); !found {
		result = updateMeta(m, result, api.KeyVisibility, cs.GetConfig(keyDefaultVisibility).(meta.Visibility).String())
	}
	cs.mxService.RUnlock()
	return result
}
func updateMeta(result, m *meta.Meta, key, val string) *meta.Meta {
	if result == nil {
		result = m.Clone()
	}
	result.Set(key, val)
	return result
}

// GetDefaultLang returns the current value of the "default-lang" key.
func (cs *configService) GetDefaultLang() string { return cs.GetConfig(keyDefaultLang).(string) }

// GetSiteName returns the current value of the "site-name" key.
func (cs *configService) GetSiteName() string { return cs.GetConfig(keySiteName).(string) }

// GetHomeZettel returns the value of the "home-zettel" key.
func (cs *configService) GetHomeZettel() id.Zid {
	homeZid := cs.GetConfig(keyHomeZettel).(id.Zid)
	if homeZid != id.Invalid {
		return homeZid
	}
	cs.mxService.RLock()
	val, _ := cs.orig.Get(keyHomeZettel)
	homeZid, _ = id.Parse(val)
	cs.mxService.RUnlock()
	return homeZid
}

// GetMaxTransclusions return the maximum number of indirect transclusions.
func (cs *configService) GetMaxTransclusions() int {
	return int(cs.GetConfig(keyMaxTransclusions).(int64))
}

// GetYAMLHeader returns the current value of the "yaml-header" key.
func (cs *configService) GetYAMLHeader() bool { return cs.GetConfig(keyYAMLHeader).(bool) }

// GetMarkerExternal returns the current value of the "marker-external" key.
func (cs *configService) GetMarkerExternal() string { return cs.GetConfig(keyMarkerExternal).(string) }

// GetFooterHTML returns HTML code that should be embedded into the footer
// of each WebUI page.
func (cs *configService) GetFooterHTML() string { return cs.GetConfig(keyFooterHTML).(string) }

// GetZettelFileSyntax returns the current value of the "zettel-file-syntax" key.
func (cs *configService) GetZettelFileSyntax() []string {
	return cs.GetConfig(keyZettelFileSyntax).([]string)
}

// --- config.AuthConfig

// GetSimpleMode returns true if system tuns in simple-mode.
func (cs *configService) GetSimpleMode() bool { return cs.GetConfig(kernel.ConfigSimpleMode).(bool) }

// GetExpertMode returns the current value of the "expert-mode" key.
func (cs *configService) GetExpertMode() bool { return cs.GetConfig(keyExpertMode).(bool) }

// GetVisibility returns the visibility value, or "login" if none is given.
func (cs *configService) GetVisibility(m *meta.Meta) meta.Visibility {
	if val, ok := m.Get(api.KeyVisibility); ok {
		if vis := meta.GetVisibility(val); vis != meta.VisibilityUnknown {
			return vis
		}
	}

	vis := cs.GetConfig(keyDefaultVisibility).(meta.Visibility)
	if vis != meta.VisibilityUnknown {
		return vis
	}
	cs.mxService.RLock()
	val, _ := cs.orig.Get(keyDefaultVisibility)
	vis = meta.GetVisibility(val)
	cs.mxService.RUnlock()
	return vis
}
