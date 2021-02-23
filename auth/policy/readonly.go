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

import "zettelstore.de/z/domain/meta"

type roPolicy struct{}

func (p *roPolicy) CanCreate(user, newMeta *meta.Meta) bool         { return false }
func (p *roPolicy) CanRead(user, m *meta.Meta) bool                 { return true }
func (p *roPolicy) CanWrite(user, oldMeta, newMeta *meta.Meta) bool { return false }
func (p *roPolicy) CanRename(user, m *meta.Meta) bool               { return false }
func (p *roPolicy) CanDelete(user, m *meta.Meta) bool               { return false }
