//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package policy provides some interfaces and implementation for authorization policies.
package policy

import (
	"zettelstore.de/z/auth"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/meta"
)

type anonPolicy struct {
	authConfig    config.AuthConfig
	getVisibility func(*meta.Meta) meta.Visibility
	pre           auth.Policy
}

func (ap *anonPolicy) CanCreate(user, newMeta *meta.Meta) bool {
	return ap.pre.CanCreate(user, newMeta)
}

func (ap *anonPolicy) CanRead(user, m *meta.Meta) bool {
	return ap.pre.CanRead(user, m) && ap.checkVisibility(m)
}

func (ap *anonPolicy) CanWrite(user, oldMeta, newMeta *meta.Meta) bool {
	return ap.pre.CanWrite(user, oldMeta, newMeta) && ap.checkVisibility(oldMeta)
}

func (ap *anonPolicy) CanRename(user, m *meta.Meta) bool {
	return ap.pre.CanRename(user, m) && ap.checkVisibility(m)
}

func (ap *anonPolicy) CanDelete(user, m *meta.Meta) bool {
	return ap.pre.CanDelete(user, m) && ap.checkVisibility(m)
}

func (ap *anonPolicy) checkVisibility(m *meta.Meta) bool {
	if ap.getVisibility(m) == meta.VisibilityExpert {
		return ap.authConfig.GetExpertMode()
	}
	return true
}
