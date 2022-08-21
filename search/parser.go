//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package search

import (
	"strconv"

	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
)

// Parse the search specification and return a Search object.
func Parse(spec string) (s *Search) { return s.Parse(spec) }

// Parse the search string and update the search object.
func (s *Search) Parse(spec string) *Search {
	state := parserState{
		inp: input.NewInput([]byte(spec)),
	}
	s = state.parse(s)
	if s != nil {
		for len(s.terms) > 1 && s.terms[len(s.terms)-1].isEmpty() {
			s.terms = s.terms[:len(s.terms)-1]
		}
	}
	return s
}

type parserState struct {
	inp *input.Input
}

func (ps *parserState) mustStop() bool { return ps.inp.Ch == input.EOS }
func (ps *parserState) acceptSingleKw(s string) bool {
	inp := ps.inp
	pos := inp.Pos
	if inp.Accept(s) && (ps.isSpace() || ps.mustStop()) {
		return true
	}
	inp.SetPos(pos)
	return false
}
func (ps *parserState) acceptKwArgs(s string) bool {
	inp := ps.inp
	pos := inp.Pos
	if inp.Accept(s) && ps.isSpace() {
		ps.skipSpace()
		return true
	}
	inp.SetPos(pos)
	return false
}

const (
	kwLimit   = "LIMIT"
	kwNegate  = "NEGATE"
	kwOffset  = "OFFSET"
	kwOr      = "OR"
	kwOrder   = "ORDER"
	kwRandom  = "RANDOM"
	kwReverse = "REVERSE"
)

func (ps *parserState) parse(sea *Search) *Search {
	inp := ps.inp
	for {
		ps.skipSpace()
		if ps.mustStop() {
			break
		}
		pos := inp.Pos
		if ps.acceptSingleKw(kwNegate) {
			sea = createIfNeeded(sea)
			sea.negate = !sea.negate
			continue
		}
		if ps.acceptSingleKw(kwOr) {
			sea = createIfNeeded(sea)
			if !sea.terms[len(sea.terms)-1].isEmpty() {
				sea.terms = append(sea.terms, conjTerms{})
			}
			continue
		}
		if ps.acceptSingleKw(kwRandom) {
			sea = createIfNeeded(sea)
			if len(sea.order) == 0 {
				sea.order = []sortOrder{{"", false}}
			}
			continue
		}
		if ps.acceptKwArgs(kwOrder) {
			if s, ok := ps.parseOrder(sea); ok {
				sea = s
				continue
			}
		}
		if ps.acceptKwArgs(kwOffset) {
			if s, ok := ps.parseOffset(sea); ok {
				sea = s
				continue
			}
		}
		if ps.acceptKwArgs(kwLimit) {
			if s, ok := ps.parseLimit(sea); ok {
				sea = s
				continue
			}
		}
		inp.SetPos(pos)
		sea = ps.parseText(sea)
	}
	return sea
}
func (ps *parserState) parseOrder(s *Search) (*Search, bool) {
	reverse := false
	if ps.acceptKwArgs(kwReverse) {
		reverse = true
	}
	word := ps.scanWord()
	if len(word) == 0 {
		return s, false
	}
	if sWord := string(word); meta.KeyIsValid(sWord) {
		s = createIfNeeded(s)
		if len(s.order) == 1 && s.order[0].isRandom() {
			s.order = nil
		}
		s.order = append(s.order, sortOrder{sWord, reverse})
		return s, true
	}
	return s, false
}

func (ps *parserState) parseOffset(s *Search) (*Search, bool) {
	num, ok := ps.scanPosInt()
	if !ok {
		return s, false
	}
	s = createIfNeeded(s)
	if s.offset <= num {
		s.offset = num
	}
	return s, true
}

func (ps *parserState) parseLimit(s *Search) (*Search, bool) {
	num, ok := ps.scanPosInt()
	if !ok {
		return s, false
	}
	s = createIfNeeded(s)
	if s.limit == 0 || s.limit >= num {
		s.limit = num
	}
	return s, true
}

func (ps *parserState) parseText(s *Search) *Search {
	pos := ps.inp.Pos
	op, hasOp := ps.scanSearchOp()
	if hasOp && (op == cmpExist || op == cmpNotExist) {
		ps.inp.SetPos(pos)
		hasOp = false
	}
	text, key := ps.scanSearchTextOrKey(hasOp)
	if len(key) > 0 {
		// Assert: hasOp == false
		op, hasOp = ps.scanSearchOp()
		// Assert hasOp == true
		if op == cmpExist || op == cmpNotExist {
			if ps.isSpace() || ps.mustStop() {
				return s.addKey(string(key), op)
			}
			ps.inp.SetPos(pos)
			hasOp = false
			text = ps.scanWord()
			key = nil
		} else {
			text = ps.scanWord()
		}
	} else if len(text) == 0 {
		// Only an empty search operation is found -> ignore it
		return s
	}
	s = createIfNeeded(s)
	if hasOp {
		if key == nil {
			s.addSearch(expValue{string(text), op})
		} else {
			last := len(s.terms) - 1
			if s.terms[last].mvals == nil {
				s.terms[last].mvals = expMetaValues{string(key): {expValue{string(text), op}}}
			} else {
				sKey := string(key)
				s.terms[last].mvals[sKey] = append(s.terms[last].mvals[sKey], expValue{string(text), op})
			}
		}
	} else {
		// Assert key == nil
		s.addSearch(expValue{string(text), cmpMatch})
	}
	return s
}

func (ps *parserState) scanSearchTextOrKey(hasOp bool) ([]byte, []byte) {
	inp := ps.inp
	pos := inp.Pos
	allowKey := !hasOp

	for !ps.isSpace() && !ps.mustStop() {
		if allowKey {
			switch inp.Ch {
			case '!', '?', ':', '=', '>', '<', '~':
				allowKey = false
				if key := inp.Src[pos:inp.Pos]; meta.KeyIsValid(string(key)) {
					return nil, key
				}
			}
		}
		inp.Next()
	}
	return inp.Src[pos:inp.Pos], nil
}

func (ps *parserState) scanWord() []byte {
	inp := ps.inp
	pos := inp.Pos
	for !ps.isSpace() && !ps.mustStop() {
		inp.Next()
	}
	return inp.Src[pos:inp.Pos]
}

func (ps *parserState) scanPosInt() (int, bool) {
	inp := ps.inp
	ch := inp.Ch
	if ch == '0' {
		ch = inp.Next()
		if isSpace(ch) || ps.mustStop() {
			return 0, true
		}
		return 0, false
	}
	word := ps.scanWord()
	if len(word) == 0 {
		return 0, false
	}
	uval, err := strconv.ParseUint(string(word), 10, 63)
	if err != nil {
		return 0, false
	}
	return int(uval), true
}

func (ps *parserState) scanSearchOp() (compareOp, bool) {
	inp := ps.inp
	ch := inp.Ch
	negate := false
	if ch == '!' {
		ch = inp.Next()
		negate = true
	}
	op := cmpUnknown
	switch ch {
	case '?':
		inp.Next()
		op = cmpExist
	case ':':
		inp.Next()
		op = cmpHas
	case '<':
		inp.Next()
		op = cmpSuffix
	case '>':
		inp.Next()
		op = cmpPrefix
	case '~':
		inp.Next()
		op = cmpMatch
	default:
		if negate {
			return cmpNoMatch, true
		}
		return cmpUnknown, false
	}
	if negate {
		return op.negate(), true
	}
	return op, true
}

func (ps *parserState) isSpace() bool {
	return isSpace(ps.inp.Ch)
}

func isSpace(ch rune) bool {
	switch ch {
	case input.EOS:
		return false
	case ' ', '\t', '\n', '\r':
		return true
	}
	return input.IsSpace(ch)
}

func (ps *parserState) skipSpace() {
	for ps.isSpace() {
		ps.inp.Next()
	}
}
