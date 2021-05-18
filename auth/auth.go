//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package auth provides services for authentification / authorization.
package auth

import (
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
)

// BaseManager allows to check some base auth modes.
type BaseManager interface {
	// IsReadonly returns true, if the systems is configured to run in read-only-mode.
	IsReadonly() bool
}

// AuthzManager provides methods for authorization.
type AuthzManager interface {
	BaseManager

	// Owner returns the zettel identifier of the owner.
	Owner() id.Zid

	// IsOwner returns true, if the given zettel identifier is that of the owner.
	IsOwner(zid id.Zid) bool

	// Returns true if authorization is enabled.
	WithAuthz() bool

	// GetUserRole role returns the user role of the given user zettel.
	GetUserRole(user *meta.Meta) meta.UserRole
}

// Manager is the main interface for providing the service.
type Manager interface {
	BaseManager
	AuthzManager

	PlaceWithPolicy(unprotectedPlace place.Place) (place.Place, Policy)
}

// Policy is an interface for checking access authorization.
type Policy interface {
	// User is allowed to create a new zettel.
	CanCreate(user, newMeta *meta.Meta) bool

	// User is allowed to read zettel
	CanRead(user, m *meta.Meta) bool

	// User is allowed to write zettel.
	CanWrite(user, oldMeta, newMeta *meta.Meta) bool

	// User is allowed to rename zettel
	CanRename(user, m *meta.Meta) bool

	// User is allowed to delete zettel
	CanDelete(user, m *meta.Meta) bool
}
