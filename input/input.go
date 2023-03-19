//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package input provides an abstraction for data to be read.
package input

import "unicode/utf8"

// Input is an abstract input source
type Input struct {
	// Read-only, will never change
	Src []byte // The source string

	// Read-only, will change
	Ch      rune // current character
	Pos     int  // character position in src
	readPos int  // reading position (position after current character)
}

// NewInput creates a new input source.
func NewInput(src []byte) *Input {
	inp := &Input{Src: src}
	inp.Next()
	return inp
}

// EOS = End of source
const EOS = rune(-1)

// Next reads the next rune into inp.Ch and returns it too.
func (inp *Input) Next() rune {
	if inp.readPos >= len(inp.Src) {
		inp.Pos = len(inp.Src)
		inp.Ch = EOS
		return EOS
	}
	inp.Pos = inp.readPos
	r, w := rune(inp.Src[inp.readPos]), 1
	if r >= utf8.RuneSelf {
		r, w = utf8.DecodeRune(inp.Src[inp.readPos:])
	}
	inp.readPos += w
	inp.Ch = r
	return r
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
			r, _ = utf8.DecodeRune(inp.Src[pos:])
		}
		if r == '\t' {
			return ' '
		}
		return r
	}
	return EOS
}

// Accept checks if the given string is a prefix of the text to be parsed.
// If successful, advance position and current character.
// String must only contain bytes < 128.
// If not successful, everything remains as it is.
func (inp *Input) Accept(s string) bool {
	pos := inp.Pos
	remaining := len(inp.Src) - pos
	if s == "" || len(s) > remaining {
		return false
	}
	// According to internal documentation of bytes.Equal, the string() will not allocate any memory.
	if readPos := pos + len(s); s == string(inp.Src[pos:readPos]) {
		inp.readPos = readPos
		inp.Next()
		return true
	}
	return false
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
	if inp.Pos != pos {
		inp.readPos = pos
		inp.Next()
	}
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

// ScanLineContent reads the reaining input stream and interprets it as lines of text.
func (inp *Input) ScanLineContent() []byte {
	result := make([]byte, 0, len(inp.Src)-inp.Pos+1)
	for {
		inp.EatEOL()
		posL := inp.Pos
		if inp.Ch == EOS {
			return result
		}
		inp.SkipToEOL()
		if len(result) > 0 {
			result = append(result, '\n')
		}
		result = append(result, inp.Src[posL:inp.Pos]...)
	}
}
