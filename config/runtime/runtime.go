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
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/stock"
)

// --- Configuration zettel --------------------------------------------------

var configStock stock.Stock

// SetupConfiguration enables the configuration data.
func SetupConfiguration(mgr place.Manager) {
	if configStock != nil {
		panic("configStock already set")
	}
	configStock = stock.NewStock(mgr)
	if err := configStock.Subscribe(id.ConfigurationZid); err != nil {
		panic(err)
	}
}

// getConfigurationMeta returns the meta data of the configuration zettel.
func getConfigurationMeta() *meta.Meta {
	if configStock == nil {
		panic("configStock not set")
	}
	return configStock.GetMeta(id.ConfigurationZid)
}

// GetDefaultTitle returns the current value of the "default-title" key.
func GetDefaultTitle() string {
	if configStock != nil {
		if config := getConfigurationMeta(); config != nil {
			if title, ok := config.Get(meta.KeyDefaultTitle); ok {
				return title
			}
		}
	}
	return "Untitled"
}

// GetDefaultSyntax returns the current value of the "default-syntax" key.
func GetDefaultSyntax() string {
	if configStock != nil {
		if config := getConfigurationMeta(); config != nil {
			if syntax, ok := config.Get(meta.KeyDefaultSyntax); ok {
				return syntax
			}
		}
	}
	return meta.ValueSyntaxZmk
}

// GetDefaultRole returns the current value of the "default-role" key.
func GetDefaultRole() string {
	if configStock != nil {
		if config := getConfigurationMeta(); config != nil {
			if role, ok := config.Get(meta.KeyDefaultRole); ok {
				return role
			}
		}
	}
	return meta.ValueRoleZettel
}

// GetDefaultLang returns the current value of the "default-lang" key.
func GetDefaultLang() string {
	if configStock != nil {
		if config := getConfigurationMeta(); config != nil {
			if lang, ok := config.Get(meta.KeyDefaultLang); ok {
				return lang
			}
		}
	}
	return "en"
}

// GetDefaultCopyright returns the current value of the "default-copyright" key.
func GetDefaultCopyright() string {
	if configStock != nil {
		if config := getConfigurationMeta(); config != nil {
			if copyright, ok := config.Get(meta.KeyDefaultCopyright); ok {
				return copyright
			}
		}
	}
	return ""
}

// GetDefaultLicense returns the current value of the "default-license" key.
func GetDefaultLicense() string {
	if configStock != nil {
		if config := getConfigurationMeta(); config != nil {
			if license, ok := config.Get(meta.KeyDefaultLicense); ok {
				return license
			}
		}
	}
	return ""
}

// GetExpertMode returns the current value of the "expert-mode" key
func GetExpertMode() bool {
	if config := getConfigurationMeta(); config != nil {
		if mode, ok := config.Get(meta.KeyExpertMode); ok {
			return meta.BoolValue(mode)
		}
	}
	return false
}

// GetSiteName returns the current value of the "site-name" key.
func GetSiteName() string {
	if config := getConfigurationMeta(); config != nil {
		if name, ok := config.Get(meta.KeySiteName); ok {
			return name
		}
	}
	return "Zettelstore"
}

// GetHomeZettel returns the value of the "home-zettel" key.
func GetHomeZettel() id.Zid {
	if config := getConfigurationMeta(); config != nil {
		if start, ok := config.Get(meta.KeyHomeZettel); ok {
			if startID, err := id.Parse(start); err == nil {
				return startID
			}
		}
	}
	return id.DefaultHomeZid
}

// GetDefaultVisibility returns the default value for zettel visibility.
func GetDefaultVisibility() meta.Visibility {
	if config := getConfigurationMeta(); config != nil {
		if value, ok := config.Get(meta.KeyDefaultVisibility); ok {
			if vis := meta.GetVisibility(value); vis != meta.VisibilityUnknown {
				return vis
			}
		}
	}
	return meta.VisibilityLogin
}

// GetYAMLHeader returns the current value of the "yaml-header" key.
func GetYAMLHeader() bool {
	if config := getConfigurationMeta(); config != nil {
		return config.GetBool(meta.KeyYAMLHeader)
	}
	return false
}

// GetZettelFileSyntax returns the current value of the "zettel-file-syntax" key.
func GetZettelFileSyntax() []string {
	if config := getConfigurationMeta(); config != nil {
		return config.GetListOrNil(meta.KeyZettelFileSyntax)
	}
	return nil
}

// GetMarkerExternal returns the current value of the "marker-external" key.
func GetMarkerExternal() string {
	if config := getConfigurationMeta(); config != nil {
		if html, ok := config.Get(meta.KeyMarkerExternal); ok {
			return html
		}
	}
	return "&#10138;"
}

// GetFooterHTML returns HTML code that should be embedded into the footer
// of each WebUI page.
func GetFooterHTML() string {
	if config := getConfigurationMeta(); config != nil {
		if data, ok := config.Get(meta.KeyFooterHTML); ok {
			return data
		}
	}
	return ""
}

// GetListPageSize returns the maximum length of a list to be returned in WebUI.
// A value less or equal to zero signals no limit.
func GetListPageSize() int {
	if config := getConfigurationMeta(); config != nil {
		if value, ok := config.GetNumber(meta.KeyListPageSize); ok && value > 0 {
			return value
		}
	}
	return 0
}
