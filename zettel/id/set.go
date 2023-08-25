//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package id

import "maps"

// Set is a set of zettel identifier
type Set map[Zid]struct{}

// NewSet returns a new set of identifier with the given initial values.
func NewSet(zids ...Zid) Set {
	l := len(zids)
	if l < 8 {
		l = 8
	}
	result := make(Set, l)
	result.CopySlice(zids)
	return result
}

// NewSetCap returns a new set of identifier with the given capacity and initial values.
func NewSetCap(c int, zids ...Zid) Set {
	l := len(zids)
	if c < l {
		c = l
	}
	if c < 8 {
		c = 8
	}
	result := make(Set, c)
	result.CopySlice(zids)
	return result
}

// Add adds a Add to the set.
func (s Set) Add(zid Zid) Set {
	if s == nil {
		return NewSet(zid)
	}
	s[zid] = struct{}{}
	return s
}

// Contains return true if the set is non-nil and the set contains the given Zettel identifier.
func (s Set) Contains(zid Zid) bool {
	if s != nil {
		_, found := s[zid]
		return found
	}
	return false
}

// ContainsOrNil return true if the set is nil or if the set contains the given Zettel identifier.
func (s Set) ContainsOrNil(zid Zid) bool {
	if s != nil {
		_, found := s[zid]
		return found
	}
	return true
}

// Copy adds all member from the other set.
func (s Set) Copy(other Set) Set {
	if s == nil {
		s = NewSetCap(len(other))
	}
	maps.Copy(s, other)
	return s
}

// CopySlice adds all identifier of the given slice to the set.
func (s Set) CopySlice(sl Slice) {
	if s == nil {
		s = NewSetCap(len(sl))
	}
	for _, zid := range sl {
		s[zid] = struct{}{}
	}
}

// Sorted returns the set as a sorted slice of zettel identifier.
func (s Set) Sorted() Slice {
	if l := len(s); l > 0 {
		result := make(Slice, 0, l)
		for zid := range s {
			result = append(result, zid)
		}
		result.Sort()
		return result
	}
	return nil
}

// IntersectOrSet removes all zettel identifier that are not in the other set.
// Both sets can be modified by this method. One of them is the set returned.
// It contains the intersection of both, if s is not nil.
//
// If s == nil, then the other set is always returned.
func (s Set) IntersectOrSet(other Set) Set {
	if s == nil {
		return other
	}
	if len(s) > len(other) {
		s, other = other, s
	}
	for zid := range s {
		_, otherOk := other[zid]
		if !otherOk {
			delete(s, zid)
		}
	}
	return s
}

// Substract removes all zettel identifier from 's' that are in the set 'other'.
func (s Set) Substract(other Set) {
	if s == nil || other == nil {
		return
	}
	for zid := range other {
		delete(s, zid)
	}
}

// IsEqual returns true if the other set is equal to the given set.
func (s Set) IsEqual(other Set) bool {
	return maps.Equal(s, other)
}
