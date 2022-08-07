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
	"bytes"

	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
)

// Parse the search specification and return a Search object.
func Parse(spec string) *Search {
	state := parserState{
		inp: input.NewInput([]byte(spec)),
	}
	return state.parse()
}

type parserState struct {
	inp *input.Input
}

const (
	kwNegate = "NEGATE"
)

func (ps *parserState) parse() *Search {
	inp := ps.inp
	var result *Search
	exprNegate := false
	for inp.Ch != input.EOS {
		ps.skipSpace()
		pos := inp.Pos
		if inp.Accept(kwNegate) && (ps.isSpace() || inp.Ch == input.EOS) {
			exprNegate = !exprNegate
			continue
		}
		inp.SetPos(pos)
		hasOp, cmpOp, cmpNegate := ps.scanSearchOp()
		text, key := ps.scanSearchTextOrKey(hasOp)
		if key != "" {
			// Assert: hasOp == false
			hasOp, cmpOp, cmpNegate = ps.scanSearchOp()
			// Assert hasOp == true
			text = ps.scanSearchText()
		} else if text == "" {
			// Only an empty search operation is found -> ignore it
			continue
		}
		if result == nil {
			result = new(Search)
		}
		if hasOp {
			result.addExpValue(key, expValue{text, cmpOp, cmpNegate})
		} else {
			// Assert key == ""
			result.addExpValue(key, expValue{text, cmpDefault, false})
		}
	}
	if exprNegate {
		result = result.SetNegate()
	}
	return result
}

func (ps *parserState) scanSearchTextOrKey(hasOp bool) (string, string) {
	inp := ps.inp
	allowKey := !hasOp

	var buf bytes.Buffer
	for !ps.isSpace() && inp.Ch != input.EOS {
		if allowKey {
			switch inp.Ch {
			case '!', ':', '=', '>', '<', '~':
				allowKey = false
				if key := buf.String(); meta.KeyIsValid(key) {
					return "", key
				}
			}
		}

		buf.WriteRune(inp.Ch)
		inp.Next()
	}
	return buf.String(), ""
}

func (ps *parserState) scanSearchText() string {
	inp := ps.inp
	var buf bytes.Buffer
	for !ps.isSpace() && inp.Ch != input.EOS {
		buf.WriteRune(inp.Ch)
		inp.Next()
	}
	return buf.String()
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
