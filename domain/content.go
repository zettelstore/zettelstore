//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package domain provides domain specific types, constants, and functions.
package domain

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"unicode"
	"unicode/utf8"

	"zettelstore.de/z/input"
)

// Content is just the content of a zettel.
type Content struct {
	data     []byte
	isBinary bool
}

// NewContent creates a new content from a string.
func NewContent(data []byte) Content {
	return Content{data: data, isBinary: calcIsBinary(data)}
}

// Equal compares two content values.
func (zc *Content) Equal(o *Content) bool {
	if zc == nil {
		return o == nil
	}
	if zc.isBinary != o.isBinary {
		return false
	}
	return bytes.Equal(zc.data, o.data)
}

// Set content to new string value.
func (zc *Content) Set(data []byte) {
	zc.data = data
	zc.isBinary = calcIsBinary(data)
}

// Write it to a Writer
func (zc *Content) Write(w io.Writer) (int, error) {
	return w.Write(zc.data)
}

// AsString returns the content itself is a string.
func (zc *Content) AsString() string { return string(zc.data) }

// AsBytes returns the content itself is a byte slice.
func (zc *Content) AsBytes() []byte { return zc.data }

// IsBinary returns true if the content contains non-unicode values or is,
// interpreted a text, with a high probability binary content.
func (zc *Content) IsBinary() bool { return zc.isBinary }

// TrimSpace remove some space character in content, if it is not binary content.
func (zc *Content) TrimSpace() {
	if zc.isBinary {
		return
	}
	inp := input.NewInput(zc.data)
	pos := inp.Pos
	for inp.Ch != input.EOS {
		if input.IsEOLEOS(inp.Ch) {
			inp.Next()
			pos = inp.Pos
			continue
		}
		if !input.IsSpace(inp.Ch) {
			break
		}
		inp.Next()
	}
	zc.data = bytes.TrimRightFunc(inp.Src[pos:], unicode.IsSpace)
}

// Encode content for future transmission.
func (zc *Content) Encode() (data, encoding string) {
	if !zc.isBinary {
		return zc.AsString(), ""
	}
	return base64.StdEncoding.EncodeToString(zc.data), "base64"
}

// SetDecoded content to the decoded value of the given string.
func (zc *Content) SetDecoded(data, encoding string) error {
	switch encoding {
	case "":
		zc.data = []byte(data)
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return err
		}
		zc.data = decoded
	default:
		return errors.New("unknown encoding " + encoding)
	}
	zc.isBinary = calcIsBinary(zc.data)
	return nil
}

func calcIsBinary(data []byte) bool {
	if !utf8.Valid(data) {
		return true
	}
	l := len(data)
	for i := 0; i < l; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}
