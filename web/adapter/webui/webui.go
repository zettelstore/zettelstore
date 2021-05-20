//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package webui provides web-UI handlers for web requests.
package webui

import (
	"zettelstore.de/z/auth"
	"zettelstore.de/z/place"
	"zettelstore.de/z/web/server"
)

// WebUI holds all data for delivering the web ui.
type WebUI struct {
	te    *templateEngine
	ab    server.AuthBuilder
	authz auth.AuthzManager
}

// New creates a new WebUI struct.
func New(ab server.AuthBuilder, authz auth.AuthzManager, token auth.TokenManager,
	mgr place.Manager, pol auth.Policy) *WebUI {
	return &WebUI{
		te:    newTemplateEngine(ab, authz, token, mgr, pol),
		ab:    ab,
		authz: authz,
	}
}
