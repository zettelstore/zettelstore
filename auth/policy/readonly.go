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

import "zettelstore.de/z/domain/meta"

type roPolicy struct{}

func (*roPolicy) CanCreate(user, newMeta *meta.Meta) bool         { return false }
func (*roPolicy) CanRead(user, m *meta.Meta) bool                 { return true }
func (*roPolicy) CanWrite(user, oldMeta, newMeta *meta.Meta) bool { return false }
func (*roPolicy) CanRename(user, m *meta.Meta) bool               { return false }
func (*roPolicy) CanDelete(user, m *meta.Meta) bool               { return false }
