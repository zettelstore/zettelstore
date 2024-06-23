//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2021-present Detlef Stern
//-----------------------------------------------------------------------------

package id

import (
	"maps"
	"strings"
)

// Set is a set of zettel identifier
type Set struct {
	m map[Zid]struct{}
}

// String returns a string representation of the map.
func (s *Set) String() string {
	if s == nil || len(s.m) == 0 {
		return "{}"
	}
	var sb strings.Builder
	sb.WriteByte('{')
	for i, zid := range s.Sorted() {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.Write(zid.Bytes())
	}
	sb.WriteByte('}')
	return sb.String()
}

// NewSet returns a new set of identifier with the given initial values.
func NewSet(zids ...Zid) *Set {
	result := Set{m: make(map[Zid]struct{}, len(zids))}
	result.CopySlice(zids)
	return &result
}

// NewSetCap returns a new set of identifier with the given capacity and initial values.
func NewSetCap(c int, zids ...Zid) *Set {
	result := Set{m: make(map[Zid]struct{}, max(c, len(zids)))}
	result.CopySlice(zids)
	return &result
}

// IsEmpty returns true, if the set conains no element.
func (s *Set) IsEmpty() bool {
	return s == nil || len(s.m) == 0
}

// Length returns the number of elements in this set.
func (s *Set) Length() int {
	if s == nil {
		return 0
	}
	return len(s.m)
}

// Clone returns a copy of the given set.
func (s *Set) Clone() *Set {
	if s == nil || len(s.m) == 0 {
		return nil
	}
	return &Set{m: maps.Clone(s.m)}
}

// Add adds a Add to the set.
func (s *Set) Add(zid Zid) *Set {
	if s == nil {
		return NewSet(zid)
	}
	s.add(zid)
	return s
}

// Contains return true if the set is non-nil and the set contains the given Zettel identifier.
func (s *Set) Contains(zid Zid) bool { return s != nil && s.contains(zid) }

// ContainsOrNil return true if the set is nil or if the set contains the given Zettel identifier.
func (s *Set) ContainsOrNil(zid Zid) bool { return s == nil || s.contains(zid) }

// Copy adds all member from the other set.
func (s *Set) Copy(other *Set) *Set {
	if s == nil {
		if other == nil || len(other.m) == 0 {
			return nil
		}
		s = NewSetCap(len(other.m))
	}
	if other != nil {
		maps.Copy(s.m, other.m)
	}
	return s
}

// CopySlice adds all identifier of the given slice to the set.
func (s *Set) CopySlice(sl Slice) *Set {
	if s == nil {
		return NewSet(sl...)
	}
	for _, zid := range sl {
		s.add(zid)
	}
	return s
}

// Sorted returns the set as a sorted slice of zettel identifier.
func (s *Set) Sorted() Slice {
	if s == nil {
		return nil
	}
	if l := len(s.m); l > 0 {
		result := make(Slice, 0, l)
		for zid := range s.m {
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
func (s *Set) IntersectOrSet(other *Set) *Set {
	if s == nil {
		return other
	}
	if other == nil {
		return nil
	}
	if len(s.m) > len(other.m) {
		s, other = other, s
	}
	for zid := range s.m {
		if !other.contains(zid) {
			s.remove(zid)
		}
	}
	return s
}

// Substract removes all zettel identifier from 's' that are in the set 'other'.
func (s *Set) Substract(other *Set) {
	if s == nil || len(s.m) == 0 || other == nil || len(other.m) == 0 {
		return
	}
	for zid := range other.m {
		s.remove(zid)
	}
}

// Remove the identifier from the set.
func (s *Set) Remove(zid Zid) *Set {
	if s == nil || len(s.m) == 0 {
		return nil
	}
	s.remove(zid)
	if len(s.m) == 0 {
		return nil
	}
	return s
}

// Equal returns true if the other set is equal to the given set.
func (s *Set) Equal(other *Set) bool {
	if s == nil {
		return other == nil
	}
	if other == nil {
		return false
	}
	return maps.Equal(s.m, other.m)
}

// ForEach calls the given function for each element of the set.
func (s *Set) ForEach(fn func(zid Zid)) {
	if s != nil {
		for zid := range s.m {
			fn(zid)
		}
	}
}

// Pop return one arbitrary element of the set.
func (s *Set) Pop() (Zid, bool) {
	if s != nil {
		for zid := range s.m {
			s.remove(zid)
			return zid, true
		}
	}
	return Invalid, false
}

// ----- unchecked base operations

func (s *Set) add(zid Zid) {
	s.m[zid] = struct{}{}
}

func (s *Set) remove(zid Zid) {
	delete(s.m, zid)
}

func (s *Set) contains(zid Zid) bool {
	_, found := s.m[zid]
	return found
}
