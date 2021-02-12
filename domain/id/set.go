//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package id provides domain specific types, constants, and functions about
// zettel identifier.
package id

// Set is a set of zettel identifier
type Set map[Zid]bool

// NewSet returns a new set of identifier with the given initial values.
func NewSet(zids ...Zid) Set {
	l := len(zids)
	if l < 8 {
		l = 8
	}
	result := make(Set, l)
	for _, zid := range zids {
		result[zid] = true
	}
	return result
}

// NewSetCap returns a new set of identifier with the given capacity and initial values.
func NewSetCap(cap int, zids ...Zid) Set {
	l := len(zids)
	if cap < l {
		cap = l
	}
	if cap < 8 {
		cap = 8
	}
	result := make(Set, cap)
	for _, zid := range zids {
		result[zid] = true
	}
	return result
}

// Sort returns the set as a sorted slice of zettel identifier.
func (s Set) Sort() []Zid {
	if l := len(s); l > 0 {
		result := make([]Zid, 0, l)
		for zid := range s {
			result = append(result, zid)
		}
		Sort(result)
		return result
	}
	return nil
}
