//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package policy provides some interfaces and implementation for authorizsation policies.
package policy

import (
	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain/meta"
)

type defaultPolicy struct{}

func (d *defaultPolicy) CanCreate(user, newMeta *meta.Meta) bool { return true }
func (d *defaultPolicy) CanRead(user, m *meta.Meta) bool         { return true }
func (d *defaultPolicy) CanWrite(user, oldMeta, newMeta *meta.Meta) bool {
	return d.canChange(user, oldMeta)
}
func (d *defaultPolicy) CanRename(user, m *meta.Meta) bool { return d.canChange(user, m) }
func (d *defaultPolicy) CanDelete(user, m *meta.Meta) bool { return d.canChange(user, m) }

func (d *defaultPolicy) canChange(user, m *meta.Meta) bool {
	metaRo, ok := m.Get(meta.KeyReadOnly)
	if !ok {
		return true
	}
	if user == nil {
		// If we are here, there is no authentication.
		// See owner.go:CanWrite.

		// No authentication: check for owner-like restriction, because the user
		// acts as an owner
		return metaRo != meta.ValueUserRoleOwner && !meta.BoolValue(metaRo)
	}

	userRole := runtime.GetUserRole(user)
	switch metaRo {
	case meta.ValueUserRoleReader:
		return userRole > meta.UserRoleReader
	case meta.ValueUserRoleWriter:
		return userRole > meta.UserRoleWriter
	case meta.ValueUserRoleOwner:
		return userRole > meta.UserRoleOwner
	}
	return !meta.BoolValue(metaRo)
}
