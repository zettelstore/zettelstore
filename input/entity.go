//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package input

import (
	"html"
	"unicode"
)

// ScanEntity scans either a named or a numbered entity and returns it as a string.
//
// For numbered entities (like &#123; or &#x123;) html.UnescapeString returns
// sometimes other values as expected, if the number is not well-formed. This
// may happen because of some strange HTML parsing rules. But these do not
// apply to Zettelmarkup. Therefore, I parse the number here in the code.
func (inp *Input) ScanEntity() (res string, success bool) {
	if inp.Ch != '&' {
		return "", false
	}
	pos := inp.Pos
	inp.Next()
	if inp.Ch == '#' {
		inp.Next()
		if inp.Ch == 'x' || inp.Ch == 'X' {
			return inp.scanEntityBase16()
		}
		return inp.scanEntityBase10()
	}
	return inp.scanEntityNamed(pos)
}

func (inp *Input) scanEntityBase16() (string, bool) {
	inp.Next()
	if inp.Ch == ';' {
		return "", false
	}
	code := 0
	for {
		switch ch := inp.Ch; ch {
		case ';':
			inp.Next()
			if r := rune(code); isValidEntity(r) {
				return string(r), true
			}
			return "", false
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			code = 16*code + int(ch-'0')
		case 'a', 'b', 'c', 'd', 'e', 'f':
			code = 16*code + int(ch-'a'+10)
		case 'A', 'B', 'C', 'D', 'E', 'F':
			code = 16*code + int(ch-'A'+10)
		default:
			return "", false
		}
		if code > unicode.MaxRune {
			return "", false
		}
		inp.Next()
	}
}

func (inp *Input) scanEntityBase10() (string, bool) {
	// Base 10 code
	if inp.Ch == ';' {
		return "", false
	}
	code := 0
	for {
		switch ch := inp.Ch; ch {
		case ';':
			inp.Next()
			if r := rune(code); isValidEntity(r) {
				return string(r), true
			}
			return "", false
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			code = 10*code + int(ch-'0')
		default:
			return "", false
		}
		if code > unicode.MaxRune {
			return "", false
		}
		inp.Next()
	}
}
func (inp *Input) scanEntityNamed(pos int) (string, bool) {
	for {
		switch inp.Ch {
		case EOS, '\n', '\r', '&':
			return "", false
		case ';':
			inp.Next()
			es := string(inp.Src[pos:inp.Pos])
			ues := html.UnescapeString(es)
			if es == ues {
				return "", false
			}
			return ues, true
		default:
			inp.Next()
		}
	}
}

// isValidEntity checks if the given code is valid for an entity.
//
// According to https://html.spec.whatwg.org/multipage/syntax.html#character-references
// ""The numeric character reference forms described above are allowed to reference any code point
// excluding U+000D CR, noncharacters, and controls other than ASCII whitespace.""
func isValidEntity(r rune) bool {
	// No C0 control and no "code point in the range U+007F DELETE to U+009F APPLICATION PROGRAM COMMAND, inclusive."
	if r < ' ' || ('\u007f' <= r && r <= '\u009f') {
		return false
	}

	// If below any noncharacter code point, return true
	//
	// See: https://infra.spec.whatwg.org/#noncharacter
	if r < '\ufdd0' {
		return true
	}

	// First range of noncharacter code points: "(...) in the range U+FDD0 to U+FDEF, inclusive"
	if r <= '\ufdef' {
		return false
	}

	// Other noncharacter code points:
	switch r {
	case '\uFFFE', '\uFFFF',
		'\U0001FFFE', '\U0001FFFF',
		'\U0002FFFE', '\U0002FFFF',
		'\U0003FFFE', '\U0003FFFF',
		'\U0004FFFE', '\U0004FFFF',
		'\U0005FFFE', '\U0005FFFF',
		'\U0006FFFE', '\U0006FFFF',
		'\U0007FFFE', '\U0007FFFF',
		'\U0008FFFE', '\U0008FFFF',
		'\U0009FFFE', '\U0009FFFF',
		'\U000AFFFE', '\U000AFFFF',
		'\U000BFFFE', '\U000BFFFF',
		'\U000CFFFE', '\U000CFFFF',
		'\U000DFFFE', '\U000DFFFF',
		'\U000EFFFE', '\U000EFFFF',
		'\U000FFFFE', '\U000FFFFF',
		'\U0010FFFE', '\U0010FFFF':
		return false
	}
	return true
}
