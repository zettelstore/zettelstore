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

import "zettelstore.de/z/auth"

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
