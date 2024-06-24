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
	"slices"
	"strings"
)

// Set is a set of zettel identifier
type Set struct {
	seq []Zid
}

// String returns a string representation of the map.
func (s *Set) String() string {
	if s == nil || len(s.seq) == 0 {
		return "{}"
	}
	var sb strings.Builder
	sb.WriteByte('{')
	for i, zid := range s.seq {
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
	switch l := len(zids); l {
	case 0:
		return &Set{seq: nil}
	case 1:
		return &Set{seq: []Zid{zids[0]}}
	default:
		result := Set{seq: make(Slice, 0, l)}
		result.AddSlice(zids)
		return &result
	}
}

// NewSetCap returns a new set of identifier with the given capacity and initial values.
func NewSetCap(c int, zids ...Zid) *Set {
	result := Set{seq: make(Slice, 0, max(c, len(zids)))}
	result.AddSlice(zids)
	return &result
}

// IsEmpty returns true, if the set conains no element.
func (s *Set) IsEmpty() bool {
	return s == nil || len(s.seq) == 0
}

// Length returns the number of elements in this set.
func (s *Set) Length() int {
	if s == nil {
		return 0
	}
	return len(s.seq)
}

// Clone returns a copy of the given set.
func (s *Set) Clone() *Set {
	if s == nil || len(s.seq) == 0 {
		return nil
	}
	return &Set{seq: slices.Clone(s.seq)}
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
		if other == nil || len(other.seq) == 0 {
			return nil
		}
		s = NewSetCap(len(other.seq))
	}
	if other != nil {
		return s.AddSlice(other.seq)
	}
	return s
}

// AddSlice adds all identifier of the given slice to the set.
func (s *Set) AddSlice(sl Slice) *Set {
	if s == nil {
		return NewSet(sl...)
	}
	s.seq = slices.Grow(s.seq, len(sl))
	for _, zid := range sl {
		s.add(zid)
	}
	return s
}

// SafeSorted returns the set as a new sorted slice of zettel identifier.
func (s *Set) SafeSorted() Slice {
	if s == nil {
		return nil
	}
	return slices.Clone(s.seq)
}

// IntersectOrSet removes all zettel identifier that are not in the other set.
// Both sets can be modified by this method. One of them is the set returned.
// It contains the intersection of both, if s is not nil.
//
// If s == nil, then the other set is always returned.
func (s *Set) IntersectOrSet(other *Set) *Set {
	if s == nil || other == nil {
		return other
	}
	topos, spos, opos := 0, 0, 0
	for spos < len(s.seq) && opos < len(other.seq) {
		sz, oz := s.seq[spos], other.seq[opos]
		if sz < oz {
			spos++
			continue
		}
		if sz > oz {
			opos++
			continue
		}
		s.seq[topos] = sz
		topos++
		spos++
		opos++
	}
	s.seq = s.seq[:topos]
	return s
}

// Substract removes all zettel identifier from 's' that are in the set 'other'.
func (s *Set) Substract(other *Set) {
	if s == nil || len(s.seq) == 0 || other == nil || len(other.seq) == 0 {
		return
	}
	topos, spos, opos := 0, 0, 0
	for spos < len(s.seq) && opos < len(other.seq) {
		sz, oz := s.seq[spos], other.seq[opos]
		if sz < oz {
			s.seq[topos] = sz
			topos++
			spos++
			continue
		}
		if sz == oz {
			spos++
		}
		opos++
	}
	for spos < len(s.seq) {
		s.seq[topos] = s.seq[spos]
		topos++
		spos++
	}
	s.seq = s.seq[:topos]
}

// Remove the identifier from the set.
func (s *Set) Remove(zid Zid) *Set {
	if s == nil || len(s.seq) == 0 {
		return nil
	}
	if pos, found := s.find(zid); found {
		copy(s.seq[pos:], s.seq[pos+1:])
		s.seq = s.seq[:len(s.seq)-1]
	}
	if len(s.seq) == 0 {
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
	return slices.Equal(s.seq, other.seq)
}

// ForEach calls the given function for each element of the set.
//
// Every element is bigger than the previous one.
func (s *Set) ForEach(fn func(zid Zid)) {
	if s != nil {
		for _, zid := range s.seq {
			fn(zid)
		}
	}
}

// Pop return one arbitrary element of the set.
func (s *Set) Pop() (Zid, bool) {
	if s != nil {
		if l := len(s.seq); l > 0 {
			zid := s.seq[l-1]
			s.seq = s.seq[:l-1]
			return zid, true
		}
	}
	return Invalid, false
}

// ----- unchecked base operations

func (s *Set) add(zid Zid) {
	if pos, found := s.find(zid); !found {
		s.seq = slices.Insert(s.seq, pos, zid)
	}
}

func (s *Set) contains(zid Zid) bool {
	_, found := s.find(zid)
	return found
}

func (s *Set) find(zid Zid) (int, bool) {
	hi := len(s.seq)
	for lo := 0; lo < hi; {
		m := lo + (hi-lo)/2
		if z := s.seq[m]; z == zid {
			return m, true
		} else if z < zid {
			lo = m + 1
		} else {
			hi = m
		}
	}
	return hi, false
}
