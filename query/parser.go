//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package query

import (
	"strconv"

	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
)

// Parse the query specification and return a Query object.
func Parse(spec string) (q *Query) { return q.Parse(spec) }

// Parse the query string and update the Query object.
func (q *Query) Parse(spec string) *Query {
	state := parserState{
		inp: input.NewInput([]byte(spec)),
	}
	q = state.parse(q)
	if q != nil {
		for len(q.terms) > 1 && q.terms[len(q.terms)-1].isEmpty() {
			q.terms = q.terms[:len(q.terms)-1]
		}
	}
	return q
}

type parserState struct {
	inp *input.Input
}

func (ps *parserState) mustStop() bool { return ps.inp.Ch == input.EOS }
func (ps *parserState) acceptSingleKw(s string) bool {
	if ps.inp.Accept(s) && (ps.isSpace() || ps.mustStop()) {
		return true
	}
	return false
}
func (ps *parserState) acceptKwArgs(s string) bool {
	if ps.inp.Accept(s) && ps.isSpace() {
		ps.skipSpace()
		return true
	}
	return false
}

const (
	actionSeparatorChar      = '|'
	existOperatorChar        = '?'
	searchOperatorNotChar    = '!'
	searchOperatorHasChar    = ':'
	searchOperatorPrefixChar = '>'
	searchOperatorSuffixChar = '<'
	searchOperatorMatchChar  = '~'

	kwLimit   = "LIMIT"
	kwOffset  = "OFFSET"
	kwOr      = "OR"
	kwOrder   = "ORDER"
	kwRandom  = "RANDOM"
	kwReverse = "REVERSE"
)

func (ps *parserState) parse(q *Query) *Query {
	inp := ps.inp
	for {
		ps.skipSpace()
		if ps.mustStop() {
			break
		}
		pos := inp.Pos
		if ps.acceptSingleKw(kwOr) {
			q = createIfNeeded(q)
			if !q.terms[len(q.terms)-1].isEmpty() {
				q.terms = append(q.terms, conjTerms{})
			}
			continue
		}
		inp.SetPos(pos)
		if ps.acceptSingleKw(kwRandom) {
			q = createIfNeeded(q)
			if len(q.order) == 0 {
				q.order = []sortOrder{{"", false}}
			}
			continue
		}
		inp.SetPos(pos)
		if ps.acceptKwArgs(kwOrder) {
			if s, ok := ps.parseOrder(q); ok {
				q = s
				continue
			}
		}
		inp.SetPos(pos)
		if ps.acceptKwArgs(kwOffset) {
			if s, ok := ps.parseOffset(q); ok {
				q = s
				continue
			}
		}
		inp.SetPos(pos)
		if ps.acceptKwArgs(kwLimit) {
			if s, ok := ps.parseLimit(q); ok {
				q = s
				continue
			}
		}
		inp.SetPos(pos)
		if isActionSep(inp.Ch) {
			q = ps.parseActions(q)
			break
		}
		q = ps.parseText(q)
	}
	return q
}

func (ps *parserState) parseOrder(q *Query) (*Query, bool) {
	reverse := false
	if ps.acceptKwArgs(kwReverse) {
		reverse = true
	}
	word := ps.scanWord()
	if len(word) == 0 {
		return q, false
	}
	if sWord := string(word); meta.KeyIsValid(sWord) {
		q = createIfNeeded(q)
		if len(q.order) == 1 && q.order[0].isRandom() {
			q.order = nil
		}
		q.order = append(q.order, sortOrder{sWord, reverse})
		return q, true
	}
	return q, false
}

func (ps *parserState) parseOffset(q *Query) (*Query, bool) {
	num, ok := ps.scanPosInt()
	if !ok {
		return q, false
	}
	q = createIfNeeded(q)
	if q.offset <= num {
		q.offset = num
	}
	return q, true
}

func (ps *parserState) parseLimit(q *Query) (*Query, bool) {
	num, ok := ps.scanPosInt()
	if !ok {
		return q, false
	}
	q = createIfNeeded(q)
	if q.limit == 0 || q.limit >= num {
		q.limit = num
	}
	return q, true
}

func (ps *parserState) parseActions(q *Query) *Query {
	ps.inp.Next()
	var words []string
	for {
		ps.skipSpace()
		word := ps.scanWord()
		if len(word) == 0 {
			break
		}
		words = append(words, string(word))
	}
	if len(words) > 0 {
		q = createIfNeeded(q)
		q.actions = words
	}
	return q
}

func (ps *parserState) parseText(q *Query) *Query {
	inp := ps.inp
	pos := inp.Pos
	op, hasOp := ps.scanSearchOp()
	if hasOp && (op == cmpExist || op == cmpNotExist) {
		inp.SetPos(pos)
		hasOp = false
	}
	text, key := ps.scanSearchTextOrKey(hasOp)
	if len(key) > 0 {
		// Assert: hasOp == false
		op, hasOp = ps.scanSearchOp()
		// Assert hasOp == true
		if op == cmpExist || op == cmpNotExist {
			if ps.isSpace() || isActionSep(inp.Ch) || ps.mustStop() {
				return q.addKey(string(key), op)
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
		return q
	}
	q = createIfNeeded(q)
	if hasOp {
		if key == nil {
			q.addSearch(expValue{string(text), op})
		} else {
			last := len(q.terms) - 1
			if q.terms[last].mvals == nil {
				q.terms[last].mvals = expMetaValues{string(key): {expValue{string(text), op}}}
			} else {
				sKey := string(key)
				q.terms[last].mvals[sKey] = append(q.terms[last].mvals[sKey], expValue{string(text), op})
			}
		}
	} else {
		// Assert key == nil
		q.addSearch(expValue{string(text), cmpMatch})
	}
	return q
}

func (ps *parserState) scanSearchTextOrKey(hasOp bool) ([]byte, []byte) {
	inp := ps.inp
	pos := inp.Pos
	allowKey := !hasOp

	for !ps.isSpace() && !isActionSep(inp.Ch) && !ps.mustStop() {
		if allowKey {
			switch inp.Ch {
			case searchOperatorNotChar, existOperatorChar, searchOperatorHasChar,
				searchOperatorPrefixChar, searchOperatorSuffixChar, searchOperatorMatchChar:
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
	for !ps.isSpace() && !isActionSep(inp.Ch) && !ps.mustStop() {
		inp.Next()
	}
	return inp.Src[pos:inp.Pos]
}

func (ps *parserState) scanPosInt() (int, bool) {
	inp := ps.inp
	ch := inp.Ch
	if ch == '0' {
		ch = inp.Next()
		if isSpace(ch) || isActionSep(inp.Ch) || ps.mustStop() {
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
	if ch == searchOperatorNotChar {
		ch = inp.Next()
		negate = true
	}
	op := cmpUnknown
	switch ch {
	case existOperatorChar:
		inp.Next()
		op = cmpExist
	case searchOperatorHasChar:
		inp.Next()
		op = cmpHas
	case searchOperatorSuffixChar:
		inp.Next()
		op = cmpSuffix
	case searchOperatorPrefixChar:
		inp.Next()
		op = cmpPrefix
	case searchOperatorMatchChar:
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

func isActionSep(ch rune) bool { return ch == actionSeparatorChar }