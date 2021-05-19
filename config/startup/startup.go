//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package startup provides functions to retrieve startup configuration data.
package startup

import (
	"strconv"
	"time"

	"zettelstore.de/z/domain/meta"
)

// Set in SetupStartupConfig
var config struct {
	withAuth      bool
	insecCookie   bool
	persistCookie bool
	htmlLifetime  time.Duration
	apiLifetime   time.Duration
}

// Predefined keys for startup zettel
const (
	KeyInsecureCookie    = "insecure-cookie"
	KeyPersistentCookie  = "persistent-cookie"
	KeyTokenLifetimeHTML = "token-lifetime-html"
	KeyTokenLifetimeAPI  = "token-lifetime-api"
)

// SetupStartupConfig initializes the startup data with content of config file.
func SetupStartupConfig(cfg *meta.Meta) {
	config.insecCookie = cfg.GetBool(KeyInsecureCookie)
	config.persistCookie = cfg.GetBool(KeyPersistentCookie)
	config.htmlLifetime = getDuration(
		cfg, KeyTokenLifetimeHTML, 1*time.Hour, 1*time.Minute, 30*24*time.Hour)
	config.apiLifetime = getDuration(
		cfg, KeyTokenLifetimeAPI, 10*time.Minute, 0, 1*time.Hour)
}

func getDuration(
	cfg *meta.Meta, key string, defDur, minDur, maxDur time.Duration) time.Duration {
	if s, ok := cfg.Get(key); ok && len(s) > 0 {
		if d, err := strconv.ParseUint(s, 10, 64); err == nil {
			secs := time.Duration(d) * time.Minute
			if secs < minDur {
				return minDur
			}
			if secs > maxDur {
				return maxDur
			}
			return secs
		}
	}
	return defDur
}

// SecureCookie returns whether the web app should set cookies to secure mode.
func SecureCookie() bool { return config.withAuth && !config.insecCookie }

// PersistentCookie returns whether the web app should set persistent cookies
// (instead of temporary).
func PersistentCookie() bool { return config.persistCookie }

// TokenLifetime return the token lifetime for the web/HTML access and for the
// API access. If lifetime for API access is equal to zero, no API access is
// possible.
func TokenLifetime() (htmlLifetime, apiLifetime time.Duration) {
	return config.htmlLifetime, config.apiLifetime
}
