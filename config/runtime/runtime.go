//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package runtime provides functions to retrieve runtime configuration data.
package runtime

import (
	"context"
	"sync"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/change"
)

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

// GetDefaultTitle returns the current value of the "default-title" key.
func GetDefaultTitle() string {
	mxConfig.RLock()
	defer mxConfig.RUnlock()
	if configMeta != nil {
		if title, ok := configMeta.Get(meta.KeyDefaultTitle); ok {
			return title
		}
	}
	return "Untitled"
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

// GetDefaultLang returns the current value of the "default-lang" key.
func GetDefaultLang() string {
	mxConfig.RLock()
	defer mxConfig.RUnlock()
	if configMeta != nil {
		if lang, ok := configMeta.Get(meta.KeyDefaultLang); ok {
			return lang
		}
	}
	return meta.ValueLangEN
}

// GetDefaultCopyright returns the current value of the "default-copyright" key.
func GetDefaultCopyright() string {
	mxConfig.RLock()
	defer mxConfig.RUnlock()
	if configMeta != nil {
		if copyright, ok := configMeta.Get(meta.KeyDefaultCopyright); ok {
			return copyright
		}
	}
	return ""
}

// GetDefaultLicense returns the current value of the "default-license" key.
func GetDefaultLicense() string {
	mxConfig.RLock()
	defer mxConfig.RUnlock()
	if configMeta != nil {
		if license, ok := configMeta.Get(meta.KeyDefaultLicense); ok {
			return license
		}
	}
	return ""
}

// GetExpertMode returns the current value of the "expert-mode" key
func GetExpertMode() bool {
	mxConfig.RLock()
	defer mxConfig.RUnlock()
	if configMeta != nil {
		if mode, ok := configMeta.Get(meta.KeyExpertMode); ok {
			return meta.BoolValue(mode)
		}
	}
	return false
}

// GetSiteName returns the current value of the "site-name" key.
func GetSiteName() string {
	mxConfig.RLock()
	defer mxConfig.RUnlock()
	if configMeta != nil {
		if name, ok := configMeta.Get(meta.KeySiteName); ok {
			return name
		}
	}
	return "Zettelstore"
}

// GetHomeZettel returns the value of the "home-zettel" key.
func GetHomeZettel() id.Zid {
	mxConfig.RLock()
	defer mxConfig.RUnlock()
	if configMeta != nil {
		if start, ok := configMeta.Get(meta.KeyHomeZettel); ok {
			if startID, err := id.Parse(start); err == nil {
				return startID
			}
		}
	}
	return id.DefaultHomeZid
}

// GetDefaultVisibility returns the default value for zettel visibility.
func GetDefaultVisibility() meta.Visibility {
	mxConfig.RLock()
	defer mxConfig.RUnlock()
	if configMeta != nil {
		if value, ok := configMeta.Get(meta.KeyDefaultVisibility); ok {
			if vis := meta.GetVisibility(value); vis != meta.VisibilityUnknown {
				return vis
			}
		}
	}
	return meta.VisibilityLogin
}

// GetYAMLHeader returns the current value of the "yaml-header" key.
func GetYAMLHeader() bool {
	mxConfig.RLock()
	defer mxConfig.RUnlock()
	if configMeta != nil {
		return configMeta.GetBool(meta.KeyYAMLHeader)
	}
	return false
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

// GetMarkerExternal returns the current value of the "marker-external" key.
func GetMarkerExternal() string {
	mxConfig.RLock()
	defer mxConfig.RUnlock()
	if configMeta != nil {
		if html, ok := configMeta.Get(meta.KeyMarkerExternal); ok {
			return html
		}
	}
	return "&#10138;"
}

// GetFooterHTML returns HTML code that should be embedded into the footer
// of each WebUI page.
func GetFooterHTML() string {
	mxConfig.RLock()
	defer mxConfig.RUnlock()
	if configMeta != nil {
		if data, ok := configMeta.Get(meta.KeyFooterHTML); ok {
			return data
		}
	}
	return ""
}

// GetListPageSize returns the maximum length of a list to be returned in WebUI.
// A value less or equal to zero signals no limit.
func GetListPageSize() int {
	mxConfig.RLock()
	defer mxConfig.RUnlock()
	if configMeta != nil {
		if value, ok := configMeta.GetNumber(meta.KeyListPageSize); ok && value > 0 {
			return value
		}
	}
	return 0
}
