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

// ZettelJSON contains all data for a zettel.
type ZettelJSON struct {
	ID       string            `json:"id"`
	URL      string            `json:"url"`
	Meta     map[string]string `json:"meta"`
	Encoding string            `json:"encoding"`
	Content  string            `json:"content"`
}

// ZettelListJSON contains all data for a list of zettel
type ZettelListJSON struct {
	List []ZettelJSON `json:"list"`
}
