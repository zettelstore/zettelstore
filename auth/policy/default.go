//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package policy

import (
	"zettelstore.de/c/api"
	"zettelstore.de/z/auth"
	"zettelstore.de/z/domain/meta"
)

type defaultPolicy struct {
	manager auth.AuthzManager
}

func (*defaultPolicy) CanCreate(_, _ *meta.Meta) bool { return true }
func (*defaultPolicy) CanRead(_, _ *meta.Meta) bool   { return true }
func (d *defaultPolicy) CanWrite(user, oldMeta, _ *meta.Meta) bool {
	return d.canChange(user, oldMeta)
}
func (d *defaultPolicy) CanRename(user, m *meta.Meta) bool { return d.canChange(user, m) }
func (d *defaultPolicy) CanDelete(user, m *meta.Meta) bool { return d.canChange(user, m) }

func (d *defaultPolicy) canChange(user, m *meta.Meta) bool {
	metaRo, ok := m.Get(api.KeyReadOnly)
	if !ok {
		return true
	}
	if user == nil {
		// If we are here, there is no authentication.
		// See owner.go:CanWrite.

		// No authentication: check for owner-like restriction, because the user
		// acts as an owner
		return metaRo != api.ValueUserRoleOwner && !meta.BoolValue(metaRo)
	}

	userRole := d.manager.GetUserRole(user)
	switch metaRo {
	case api.ValueUserRoleReader:
		return userRole > meta.UserRoleReader
	case api.ValueUserRoleWriter:
		return userRole > meta.UserRoleWriter
	case api.ValueUserRoleOwner:
		return userRole > meta.UserRoleOwner
	}
	return !meta.BoolValue(metaRo)
}
