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

// Supported part values.
const (
	PartID      = "id"
	PartMeta    = "meta"
	PartContent = "content"
	PartZettel  = "zettel"
)
