//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package impl provides services for authentification / authorization.
package impl

import (
	"zettelstore.de/z/auth"
	"zettelstore.de/z/auth/policy"
	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
)

type myAuth struct {
	readonly bool
	owner    id.Zid
}

// New creates a new auth object.
func New(readonly bool, owner id.Zid) auth.Manager {
	return &myAuth{
		readonly: readonly,
		owner:    owner,
	}
}

// IsReadonly returns true, if the systems is configured to run in read-only-mode.
func (a *myAuth) IsReadonly() bool { return a.readonly }

func (a *myAuth) Owner() id.Zid { return a.owner }

func (a *myAuth) IsOwner(zid id.Zid) bool {
	return zid.IsValid() && zid == a.owner
}

func (a *myAuth) WithAuth() bool { return a.owner != id.Invalid }

// GetUserRole role returns the user role of the given user zettel.
func (a *myAuth) GetUserRole(user *meta.Meta) meta.UserRole {
	if user == nil {
		if a.WithAuth() {
			return meta.UserRoleUnknown
		}
		return meta.UserRoleOwner
	}
	if a.IsOwner(user.Zid) {
		return meta.UserRoleOwner
	}
	if val, ok := user.Get(meta.KeyUserRole); ok {
		if ur := meta.GetUserRole(val); ur != meta.UserRoleUnknown {
			return ur
		}
	}
	return meta.UserRoleReader
}

func (a *myAuth) PlaceWithPolicy(unprotectedPlace place.Place) (place.Place, auth.Policy) {
	return policy.PlaceWithPolicy(
		a, unprotectedPlace, runtime.GetExpertMode, runtime.GetVisibility)
}
