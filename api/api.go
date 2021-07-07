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

// ZidJSON contains the identifier data of a zettel.
type ZidJSON struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// ZettelDataJSON contains all data for a zettel.
type ZettelDataJSON struct {
	Meta     map[string]string `json:"meta"`
	Encoding string            `json:"encoding"`
	Content  string            `json:"content"`
}

// ZettelJSON contains all data for a zettel, the identifier, the metadata, and the content.
type ZettelJSON struct {
	ZidJSON
	ZettelDataJSON
}

// ZettelListJSON contains all data for a list of zettel
type ZettelListJSON struct {
	List []ZettelJSON `json:"list"`
}

// TagListJSON specifies the list/map of tags
type TagListJSON struct {
	Tags map[string][]string `json:"tags"`
}

// RoleListJSON specifies the list of roles.
type RoleListJSON struct {
	Roles []string `json:"role-list"`
}
