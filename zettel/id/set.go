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

// SetO is a set of zettel identifier
type SetO struct {
	seq []ZidO
}

// String returns a string representation of the set.
func (s *SetO) String() string {
	return "{" + s.MetaString() + "}"
}

// MetaString returns a string representation of the set to be stored as metadata.
func (s *SetO) MetaString() string {
	if s == nil || len(s.seq) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, zid := range s.seq {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.Write(zid.Bytes())
	}
	return sb.String()
}

// NewSetO returns a new set of identifier with the given initial values.
func NewSetO(zids ...ZidO) *SetO {
	switch l := len(zids); l {
	case 0:
		return &SetO{seq: nil}
	case 1:
		return &SetO{seq: []ZidO{zids[0]}}
	default:
		result := SetO{seq: make([]ZidO, 0, l)}
		result.AddSlice(zids)
		return &result
	}
}

// NewSetCapO returns a new set of identifier with the given capacity and initial values.
func NewSetCapO(c int, zids ...ZidO) *SetO {
	result := SetO{seq: make(SliceO, 0, max(c, len(zids)))}
	result.AddSlice(zids)
	return &result
}

// IsEmpty returns true, if the set conains no element.
func (s *SetO) IsEmpty() bool {
	return s == nil || len(s.seq) == 0
}

// Length returns the number of elements in this set.
func (s *SetO) Length() int {
	if s == nil {
		return 0
	}
	return len(s.seq)
}

// Clone returns a copy of the given set.
func (s *SetO) Clone() *SetO {
	if s == nil || len(s.seq) == 0 {
		return nil
	}
	return &SetO{seq: slices.Clone(s.seq)}
}

// Add adds a zid to the set.
func (s *SetO) Add(zid ZidO) *SetO {
	if s == nil {
		return NewSetO(zid)
	}
	s.add(zid)
	return s
}

// Contains return true if the set is non-nil and the set contains the given Zettel identifier.
func (s *SetO) Contains(zid ZidO) bool { return s != nil && s.contains(zid) }

// ContainsOrNil return true if the set is nil or if the set contains the given Zettel identifier.
func (s *SetO) ContainsOrNil(zid ZidO) bool { return s == nil || s.contains(zid) }

// AddSlice adds all identifier of the given slice to the set.
func (s *SetO) AddSlice(sl SliceO) *SetO {
	if s == nil {
		return NewSetO(sl...)
	}
	s.seq = slices.Grow(s.seq, len(sl))
	for _, zid := range sl {
		s.add(zid)
	}
	return s
}

// SafeSorted returns the set as a new sorted slice of zettel identifier.
func (s *SetO) SafeSorted() SliceO {
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
func (s *SetO) IntersectOrSet(other *SetO) *SetO {
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

// IUnion adds the elements of set other to s.
func (s *SetO) IUnion(other *SetO) *SetO {
	if other == nil || len(other.seq) == 0 {
		return s
	}
	// TODO: if other is large enough (and s is not too small) -> optimize by swapping and/or loop through both
	return s.AddSlice(other.seq)
}

// ISubstract removes all zettel identifier from 's' that are in the set 'other'.
func (s *SetO) ISubstract(other *SetO) {
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

// Diff returns the difference sets between the two sets: the first difference
// set is the set of elements that are in other, but not in s; the second
// difference set is the set of element that are in s but not in other.
//
// in other words: the first result is the set of elements from other that must
// be added to s; the second result is the set of elements that must be removed
// from s, so that s would have the same elemest as other.
func (s *SetO) Diff(other *SetO) (newS, remS *SetO) {
	if s == nil || len(s.seq) == 0 {
		return other.Clone(), nil
	}
	if other == nil || len(other.seq) == 0 {
		return nil, s.Clone()
	}
	seqS, seqO := s.seq, other.seq
	var newRefs, remRefs SliceO
	npos, opos := 0, 0
	for npos < len(seqO) && opos < len(seqS) {
		rn, ro := seqO[npos], seqS[opos]
		if rn == ro {
			npos++
			opos++
			continue
		}
		if rn < ro {
			newRefs = append(newRefs, rn)
			npos++
			continue
		}
		remRefs = append(remRefs, ro)
		opos++
	}
	if npos < len(seqO) {
		newRefs = append(newRefs, seqO[npos:]...)
	}
	if opos < len(seqS) {
		remRefs = append(remRefs, seqS[opos:]...)
	}
	return newFromSlice(newRefs), newFromSlice(remRefs)
}

// Remove the identifier from the set.
func (s *SetO) Remove(zid ZidO) *SetO {
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
func (s *SetO) Equal(other *SetO) bool {
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
func (s *SetO) ForEach(fn func(zid ZidO)) {
	if s != nil {
		for _, zid := range s.seq {
			fn(zid)
		}
	}
}

// Pop return one arbitrary element of the set.
func (s *SetO) Pop() (ZidO, bool) {
	if s != nil {
		if l := len(s.seq); l > 0 {
			zid := s.seq[l-1]
			s.seq = s.seq[:l-1]
			return zid, true
		}
	}
	return InvalidO, false
}

// Optimize the amount of memory to store the set.
func (s *SetO) Optimize() {
	if s != nil {
		s.seq = slices.Clip(s.seq)
	}
}

// ----- unchecked base operations

func newFromSlice(seq SliceO) *SetO {
	if l := len(seq); l == 0 {
		return nil
	} else {
		return &SetO{seq: seq}
	}
}

func (s *SetO) add(zid ZidO) {
	if pos, found := s.find(zid); !found {
		s.seq = slices.Insert(s.seq, pos, zid)
	}
}

func (s *SetO) contains(zid ZidO) bool {
	_, found := s.find(zid)
	return found
}

func (s *SetO) find(zid ZidO) (int, bool) {
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
