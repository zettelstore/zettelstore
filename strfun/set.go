//-----------------------------------------------------------------------------
// Copyright (c) 2022-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2022-present Detlef Stern
//-----------------------------------------------------------------------------

package strfun

// Set ist a set of strings.
type Set map[string]struct{}

// NewSet creates a new set from the given values.
func NewSet(values ...string) Set {
	s := make(Set, len(values))
	for _, v := range values {
		s.Set(v)
	}
	return s
}

// Set adds the given string to the set.
func (s Set) Set(v string) { s[v] = struct{}{} }

// Has returns true, if given value is in set.
func (s Set) Has(v string) bool { _, found := s[v]; return found }
