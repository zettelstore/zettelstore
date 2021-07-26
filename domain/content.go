//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package domain provides domain specific types, constants, and functions.
package domain

import (
	"encoding/base64"
	"errors"
	"io"
	"unicode/utf8"

	"zettelstore.de/z/strfun"
)

// Content is just the content of a zettel.
type Content struct {
	data     string
	isBinary bool
}

// NewContent creates a new content from a string.
func NewContent(s string) Content {
	return Content{data: s, isBinary: calcIsBinary(s)}
}

// Set content to new string value.
func (zc *Content) Set(s string) {
	zc.data = s
	zc.isBinary = calcIsBinary(s)
}

// Write it to a Writer
func (zc *Content) Write(w io.Writer) (int, error) {
	return io.WriteString(w, zc.data)
}

// AsString returns the content itself is a string.
func (zc *Content) AsString() string { return zc.data }

// AsBytes returns the content itself is a byte slice.
func (zc *Content) AsBytes() []byte { return []byte(zc.data) }

// IsBinary returns true if the content contains non-unicode values or is,
// interpreted a text, with a high probability binary content.
func (zc *Content) IsBinary() bool { return zc.isBinary }

// TrimSpace remove some space character in content, if it is not binary content.
func (zc *Content) TrimSpace() {
	if zc.isBinary {
		return
	}
	zc.data = strfun.TrimSpaceRight(zc.data)
}

// Encode content for future transmission.
func (zc *Content) Encode() (data, encoding string) {
	if !zc.isBinary {
		return zc.data, ""
	}
	return base64.StdEncoding.EncodeToString([]byte(zc.data)), "base64"
}

// SetDecoded content to the decoded value of the given string.
func (zc *Content) SetDecoded(data, encoding string) error {
	switch encoding {
	case "":
		zc.data = data
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return err
		}
		zc.data = string(decoded)
	default:
		return errors.New("unknown encoding " + encoding)
	}
	zc.isBinary = calcIsBinary(zc.data)
	return nil
}

func calcIsBinary(s string) bool {
	if !utf8.ValidString(s) {
		return true
	}
	l := len(s)
	for i := 0; i < l; i++ {
		if s[i] == 0 {
			return true
		}
	}
	return false
}
