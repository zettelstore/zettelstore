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
	ID string `json:"id"`
}

// ZidMetaJSON contains the identifier and the metadata of a zettel.
type ZidMetaJSON struct {
	ID   string            `json:"id"`
	Meta map[string]string `json:"meta"`
}

// ZidMetaRelatedList contains identifier/metadata of a zettel and the same for related zettel
type ZidMetaRelatedList struct {
	ID   string            `json:"id"`
	Meta map[string]string `json:"meta"`
	List []ZidMetaJSON     `json:"list"`
}

// ZettelLinksJSON store all links / connections from one zettel to other.
type ZettelLinksJSON struct {
	ID     string `json:"id"`
	Linked struct {
		Incoming []string `json:"incoming,omitempty"`
		Outgoing []string `json:"outgoing,omitempty"`
		Local    []string `json:"local,omitempty"`
		External []string `json:"external,omitempty"`
		Meta     []string `json:"meta,omitempty"`
	} `json:"linked"`
	Embedded struct {
		Outgoing []string `json:"outgoing,omitempty"`
		Local    []string `json:"local,omitempty"`
		External []string `json:"external,omitempty"`
	} `json:"embedded,omitempty"`
	Cites []string `json:"cites,omitempty"`
}

// ZettelDataJSON contains all data for a zettel.
type ZettelDataJSON struct {
	Meta     map[string]string `json:"meta"`
	Encoding string            `json:"encoding"`
	Content  string            `json:"content"`
}

// ZettelJSON contains all data for a zettel, the identifier, the metadata, and the content.
type ZettelJSON struct {
	ID       string            `json:"id"`
	Meta     map[string]string `json:"meta"`
	Encoding string            `json:"encoding"`
	Content  string            `json:"content"`
}

// ZettelListJSON contains data for a zettel list.
type ZettelListJSON struct {
	List []ZidMetaJSON `json:"list"`
}

// TagListJSON specifies the list/map of tags
type TagListJSON struct {
	Tags map[string][]string `json:"tags"`
}

// RoleListJSON specifies the list of roles.
type RoleListJSON struct {
	Roles []string `json:"role-list"`
}
