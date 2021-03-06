//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package api contains common definition used for client and server.
package api

import (
	"fmt"
)

// Additional HTTP constants used.
const (
	MethodMove = "MOVE" // HTTP method for renaming a zettel

	HeaderAccept      = "Accept"
	HeaderContentType = "Content-Type"
	HeaderDestination = "Destination"
	HeaderLocation    = "Location"
)

// Values for HTTP query parameter.
const (
	QueryKeyDepth  = "depth"
	QueryKeyDir    = "dir"
	QueryKeyFormat = "_format"
	QueryKeyLimit  = "limit"
	QueryKeyPart   = "_part"
)

// Supported dir values.
const (
	DirBackward = "backward"
	DirForward  = "forward"
)

// Supported format values.
const (
	FormatDJSON  = "djson"
	FormatHTML   = "html"
	FormatJSON   = "json"
	FormatNative = "native"
	FormatRaw    = "raw"
	FormatText   = "text"
	FormatZMK    = "zmk"
)

var formatEncoder = map[string]EncodingEnum{
	FormatDJSON:  EncoderDJSON,
	FormatHTML:   EncoderHTML,
	FormatJSON:   EncoderJSON,
	FormatNative: EncoderNative,
	FormatRaw:    EncoderRaw,
	FormatText:   EncoderText,
	FormatZMK:    EncoderZmk,
}
var encoderFormat = map[EncodingEnum]string{}

func init() {
	for k, v := range formatEncoder {
		encoderFormat[v] = k
	}
}

// Encoder returns the internal encoder code for the given format string.
func Encoder(format string) EncodingEnum {
	if e, ok := formatEncoder[format]; ok {
		return e
	}
	return EncoderUnknown
}

// EncodingEnum lists all valid encoder keys.
type EncodingEnum uint8

// Values for EncoderEnum
const (
	EncoderUnknown EncodingEnum = iota
	EncoderDJSON
	EncoderHTML
	EncoderJSON
	EncoderNative
	EncoderRaw
	EncoderText
	EncoderZmk
)

// String representation of an encoder key.
func (e EncodingEnum) String() string {
	if f, ok := encoderFormat[e]; ok {
		return f
	}
	return fmt.Sprintf("*Unknown*(%d)", e)
}

// Supported part values.
const (
	PartID      = "id"
	PartMeta    = "meta"
	PartContent = "content"
	PartZettel  = "zettel"
)
