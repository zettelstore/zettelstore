//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package input provides an abstraction for data to be read.
package input

import (
	"html"
	"unicode"
	"unicode/utf8"
)

// Input is an abstract input source
type Input struct {
	// Read-only, will never change
	Src string // The source string

	// Read-only, will change
	Ch      rune // current character
	Pos     int  // character position in src
	readPos int  // reading position (position after current character)
}

// NewInput creates a new input source.
func NewInput(src string) *Input {
	inp := &Input{Src: src}
	inp.Next()
	return inp
}

// EOS = End of source
const EOS = rune(-1)

// Next reads the next rune into inp.Ch.
func (inp *Input) Next() {
	if inp.readPos < len(inp.Src) {
		inp.Pos = inp.readPos
		r, w := rune(inp.Src[inp.readPos]), 1
		if r >= utf8.RuneSelf {
			r, w = utf8.DecodeRuneInString(inp.Src[inp.readPos:])
		}
		inp.readPos += w
		inp.Ch = r
	} else {
		inp.Pos = len(inp.Src)
		inp.Ch = EOS
	}
}

// Peek returns the rune following the most recently read rune without
// advancing. If end-of-source was already found peek returns EOS.
func (inp *Input) Peek() rune {
	return inp.PeekN(0)
}

// PeekN returns the n-th rune after the most recently read rune without
// advancing. If end-of-source was already found peek returns EOS.
func (inp *Input) PeekN(n int) rune {
	pos := inp.readPos + n
	if pos < len(inp.Src) {
		r := rune(inp.Src[pos])
		if r >= utf8.RuneSelf {
			r, _ = utf8.DecodeRuneInString(inp.Src[pos:])
		}
		if r == '\t' {
			return ' '
		}
		return r
	}
	return EOS
}

// IsEOLEOS returns true if char is either EOS or EOL.
func IsEOLEOS(ch rune) bool {
	switch ch {
	case EOS, '\n', '\r':
		return true
	}
	return false
}

// EatEOL transforms both "\r" and "\r\n" into "\n".
func (inp *Input) EatEOL() {
	switch inp.Ch {
	case '\r':
		if inp.Peek() == '\n' {
			inp.Next()
		}
		inp.Ch = '\n'
		inp.Next()
	case '\n':
		inp.Next()
	}
}

// SetPos allows to reset the read position.
func (inp *Input) SetPos(pos int) {
	inp.readPos = pos
	inp.Next()
}

// SkipToEOL reads until the next end-of-line.
func (inp *Input) SkipToEOL() {
	for {
		switch inp.Ch {
		case EOS, '\n', '\r':
			return
		}
		inp.Next()
	}
}

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
			return string(rune(code)), true
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
			return string(rune(code)), true
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
		case EOS, '\n', '\r':
			return "", false
		case ';':
			inp.Next()
			es := inp.Src[pos:inp.Pos]
			ues := html.UnescapeString(es)
			if es == ues {
				return "", false
			}
			return ues, true
		}
		inp.Next()
	}
}
