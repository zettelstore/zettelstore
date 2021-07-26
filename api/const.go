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

import "fmt"

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
	QueryKeyDepth    = "depth"
	QueryKeyDir      = "dir"
	QueryKeyEncoding = "_enc"
	QueryKeyLimit    = "limit"
	QueryKeyPart     = "_part"
	QueryKeyRaw      = "raw"
)

// Supported dir values.
const (
	DirBackward = "backward"
	DirForward  = "forward"
)

// Supported encoding values.
const (
	EncodingDJSON  = "djson"
	EncodingHTML   = "html"
	EncodingJSON   = "json"
	EncodingNative = "native"
	EncodingRaw    = "raw"
	EncodingText   = "text"
	EncodingZMK    = "zmk"
)

var mapEncodingEnum = map[string]EncodingEnum{
	EncodingDJSON:  EncoderDJSON,
	EncodingHTML:   EncoderHTML,
	EncodingJSON:   EncoderJSON,
	EncodingNative: EncoderNative,
	EncodingRaw:    EncoderRaw,
	EncodingText:   EncoderText,
	EncodingZMK:    EncoderZmk,
}
var mapEnumEncoding = map[EncodingEnum]string{}

func init() {
	for k, v := range mapEncodingEnum {
		mapEnumEncoding[v] = k
	}
}

// Encoder returns the internal encoder code for the given encoding string.
func Encoder(encoding string) EncodingEnum {
	if e, ok := mapEncodingEnum[encoding]; ok {
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
	if f, ok := mapEnumEncoding[e]; ok {
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
