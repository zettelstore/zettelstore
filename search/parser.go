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

func (ps *parserState) parse() *Search {
	var result *Search
	for ps.inp.Ch != input.EOS {
		ps.skipSpace()
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
	return result
}

func (ps *parserState) scanSearchTextOrKey(hasOp bool) (string, string) {
	inp := ps.inp
	startPos := inp.Pos
	foundOp := hasOp

	for inp.Ch != input.EOS {
		if ps.isSpace() {
			break
		}
		if !foundOp {
			switch inp.Ch {
			case '!', ':', '=', '>', '<', '~':
				foundOp = true
				if key := string(inp.Src[startPos:inp.Pos]); meta.KeyIsValid(key) {
					return "", key
				}
			}
		}
		inp.Next()
	}
	return string(inp.Src[startPos:inp.Pos]), ""
}

func (ps *parserState) scanSearchText() string {
	inp := ps.inp
	startPos := inp.Pos
	for !ps.isSpace() && inp.Ch != input.EOS {
		inp.Next()
	}
	return string(inp.Src[startPos:inp.Pos])
}

func (ps *parserState) scanSearchOp() (bool, compareOp, bool) {
	inp := ps.inp
	negate := false
	if inp.Ch == '!' {
		inp.Next()
		negate = true
	}
	switch inp.Ch {
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
	switch ps.inp.Ch {
	case input.EOS:
		return false
	case ' ', '\t', '\n', '\r':
		return true
	}
	return input.IsSpace(ps.inp.Ch)
}

func (ps *parserState) skipSpace() {
	for ps.isSpace() {
		ps.inp.Next()
	}
}
