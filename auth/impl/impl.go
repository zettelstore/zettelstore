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
	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/place"
)

type myAuth struct {
	readonly bool
}

// New creates a new auth object.
func New(readonly bool) auth.Manager {
	return &myAuth{
		readonly: readonly,
	}
}

// IsReadonly returns true, if the systems is configured to run in read-only-mode.
func (a *myAuth) IsReadonly() bool { return a.readonly }

func (a *myAuth) PlaceWithPolicy(unprotectedPlace place.Place) (place.Place, auth.Policy) {
	return policy.PlaceWithPolicy(
		unprotectedPlace, startup.WithAuth, a.IsReadonly(), runtime.GetExpertMode,
		startup.IsOwner, runtime.GetVisibility)
}
