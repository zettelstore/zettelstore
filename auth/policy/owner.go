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
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

type ownerPolicy struct {
	expertMode    func() bool
	isOwner       func(id.Zid) bool
	getVisibility func(*meta.Meta) meta.Visibility
	pre           Policy
}

func (o *ownerPolicy) CanCreate(user, newMeta *meta.Meta) bool {
	if user == nil || !o.pre.CanCreate(user, newMeta) {
		return false
	}
	return o.userIsOwner(user) || o.userCanCreate(user, newMeta)
}

func (o *ownerPolicy) userCanCreate(user *meta.Meta, newMeta *meta.Meta) bool {
	if runtime.GetUserRole(user) == meta.UserRoleReader {
		return false
	}
	if role, ok := newMeta.Get(meta.KeyRole); ok && role == meta.ValueRoleUser {
		return false
	}
	return true
}

func (o *ownerPolicy) CanRead(user, m *meta.Meta) bool {
	// No need to call o.pre.CanRead(user, meta), because it will always return true.
	// Both the default and the readonly policy allow to read a zettel.
	vis := o.getVisibility(m)
	if res, ok := o.checkVisibility(user, vis); ok {
		return res
	}
	return o.userIsOwner(user) || o.userCanRead(user, m, vis)
}

func (o *ownerPolicy) userCanRead(user, m *meta.Meta, vis meta.Visibility) bool {
	switch vis {
	case meta.VisibilityOwner, meta.VisibilitySimple, meta.VisibilityExpert:
		return false
	case meta.VisibilityPublic:
		return true
	}
	if user == nil {
		return false
	}
	if role, ok := m.Get(meta.KeyRole); ok && role == meta.ValueRoleUser {
		// Only the user can read its own zettel
		return user.Zid == m.Zid
	}
	return true
}

var noChangeUser = []string{
	meta.KeyID,
	meta.KeyRole,
	meta.KeyUserID,
	meta.KeyUserRole,
}

func (o *ownerPolicy) CanWrite(user, oldMeta, newMeta *meta.Meta) bool {
	if user == nil || !o.pre.CanWrite(user, oldMeta, newMeta) {
		return false
	}
	vis := o.getVisibility(oldMeta)
	if res, ok := o.checkVisibility(user, vis); ok {
		return res
	}
	if o.userIsOwner(user) {
		return true
	}
	if !o.userCanRead(user, oldMeta, vis) {
		return false
	}
	if role, ok := oldMeta.Get(meta.KeyRole); ok && role == meta.ValueRoleUser {
		// Here we know, that user.Zid == newMeta.Zid (because of userCanRead) and
		// user.Zid == newMeta.Zid (because oldMeta.Zid == newMeta.Zid)
		for _, key := range noChangeUser {
			if oldMeta.GetDefault(key, "") != newMeta.GetDefault(key, "") {
				return false
			}
		}
		return true
	}
	if runtime.GetUserRole(user) == meta.UserRoleReader {
		return false
	}
	return o.userCanCreate(user, newMeta)
}

func (o *ownerPolicy) CanRename(user, m *meta.Meta) bool {
	if user == nil || !o.pre.CanRename(user, m) {
		return false
	}
	if res, ok := o.checkVisibility(user, o.getVisibility(m)); ok {
		return res
	}
	return o.userIsOwner(user)
}

func (o *ownerPolicy) CanDelete(user, m *meta.Meta) bool {
	if user == nil || !o.pre.CanDelete(user, m) {
		return false
	}
	if res, ok := o.checkVisibility(user, o.getVisibility(m)); ok {
		return res
	}
	return o.userIsOwner(user)
}

func (o *ownerPolicy) checkVisibility(user *meta.Meta, vis meta.Visibility) (bool, bool) {
	switch vis {
	case meta.VisibilitySimple, meta.VisibilityExpert:
		return o.userIsOwner(user) && o.expertMode(), true
	}
	return false, false
}

func (o *ownerPolicy) userIsOwner(user *meta.Meta) bool {
	if user == nil {
		return false
	}
	if o.isOwner(user.Zid) {
		return true
	}
	if val, ok := user.Get(meta.KeyUserRole); ok && val == meta.ValueUserRoleOwner {
		return true
	}
	return false
}

func (o *ownerPolicy) CanReload(user *meta.Meta) bool {
	if !o.pre.CanReload(user) && !o.expertMode() {
		return false
	}

	// Only the owner is allowed to reload a place
	return o.userIsOwner(user)
}
