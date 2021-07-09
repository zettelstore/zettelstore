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

	"zettelstore.de/z/encoder"
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
	QueryKeyFormat = "_format"
	QueryKeyPart   = "_part"
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

var formatEncoder = map[string]encoder.Enum{
	FormatDJSON:  encoder.EncoderDJSON,
	FormatHTML:   encoder.EncoderHTML,
	FormatJSON:   encoder.EncoderJSON,
	FormatNative: encoder.EncoderNative,
	FormatRaw:    encoder.EncoderRaw,
	FormatText:   encoder.EncoderText,
	FormatZMK:    encoder.EncoderZmk,
}
var encoderFormat = map[encoder.Enum]string{}

func init() {
	for k, v := range formatEncoder {
		encoderFormat[v] = k
	}
}

// Encoder returns the internal encoder code for the given format string.
func Encoder(format string) encoder.Enum {
	if e, ok := formatEncoder[format]; ok {
		return e
	}
	return encoder.EncoderUnknown
}

// Format returns the API format of the given encoder
func Format(e encoder.Enum) string {
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
