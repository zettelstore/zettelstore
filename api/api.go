//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package api contains common definition used for client and server.
package api

// AuthJSON contains the result of an authentication call.
type AuthJSON struct {
	Token   string `json:"token"`
	Type    string `json:"token_type"`
	Expires int    `json:"expires_in"`
}
