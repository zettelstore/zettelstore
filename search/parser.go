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

// StopParsePred is a predicate that signals when parsing must end.
type StopParsePred func(*input.Input) bool

// Parse the search specification and return a Search object.
func Parse(spec string, stop StopParsePred) *Search {
	state := parserState{
		inp:  input.NewInput([]byte(spec)),
		stop: stop,
	}
	return state.parse()
}

type parserState struct {
	inp  *input.Input
	stop StopParsePred
}

func (ps *parserState) mustStop() bool {
	inp := ps.inp
	return inp.Ch == input.EOS || (ps.stop != nil && ps.stop(inp))
}

const (
	kwLimit   = "LIMIT"
	kwNegate  = "NEGATE"
	kwOffset  = "OFFSET"
	kwOrder   = "ORDER"
	kwRandom  = "RANDOM"
	kwReverse = "REVERSE"
)

func (ps *parserState) parse() *Search {
	inp := ps.inp
	var result *Search
	for {
		ps.skipSpace()
		if ps.mustStop() {
			break
		}
		pos := inp.Pos
		if inp.Accept(kwNegate) && (ps.isSpace() || ps.mustStop()) {
			result = createIfNeeded(result)
			result.negate = !result.negate
			continue
		}
		if inp.Accept(kwRandom) && (ps.isSpace() || ps.mustStop()) {
			result = createIfNeeded(result)
			if len(result.order) == 0 {
				result.order = []sortOrder{{"", false}}
			}
			continue
		}
		if inp.Accept(kwOrder) && ps.isSpace() {
			ps.skipSpace()
			if s, ok := ps.parseOrder(result); ok {
				result = s
				continue
			}
		}
		if inp.Accept(kwOffset) && ps.isSpace() {
			ps.skipSpace()
			if s, ok := ps.parseOffset(result); ok {
				result = s
				continue
			}
		}
		if inp.Accept(kwLimit) && ps.isSpace() {
			ps.skipSpace()
			if s, ok := ps.parseLimit(result); ok {
				result = s
				continue
			}
		}
		inp.SetPos(pos)
		result = ps.parseText(result)
	}
	return result
}
func (ps *parserState) parseOrder(s *Search) (*Search, bool) {
	reverse := false
	if ps.inp.Accept(kwReverse) && ps.isSpace() {
		reverse = true
		ps.skipSpace()
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
	hasOp, cmpOp, cmpNegate := ps.scanSearchOp()
	text, key := ps.scanSearchTextOrKey(hasOp)
	if key != nil {
		// Assert: hasOp == false
		hasOp, cmpOp, cmpNegate = ps.scanSearchOp()
		// Assert hasOp == true
		text = ps.scanWord()
	} else if text == nil {
		// Only an empty search operation is found -> ignore it
		return s
	}
	s = createIfNeeded(s)
	if hasOp {
		s.addExpValue(string(key), expValue{string(text), cmpOp, cmpNegate})
	} else {
		// Assert key == nil
		s.addExpValue("", expValue{string(text), cmpDefault, false})
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
			case '!', ':', '=', '>', '<', '~':
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

func (ps *parserState) scanSearchOp() (bool, compareOp, bool) {
	inp := ps.inp
	ch := inp.Ch
	negate := false
	if ch == '!' {
		ch = inp.Next()
		negate = true
	}
	switch ch {
	case ':':
		inp.Next()
		return true, cmpDefault, negate
	case '=':
		inp.Next()
		return true, cmpEqual, negate
	case '<':
		inp.Next()
		return true, cmpSuffix, negate
	case '>':
		inp.Next()
		return true, cmpPrefix, negate
	case '~':
		inp.Next()
		return true, cmpContains, negate
	}
	if negate {
		return true, cmpDefault, true
	}
	return false, cmpUnknown, false
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
