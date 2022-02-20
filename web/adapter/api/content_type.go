//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package api provides api handlers for web requests.
package api

import "zettelstore.de/c/api"

const (
	ctHTML      = "text/html; charset=utf-8"
	ctJSON      = "application/json"
	ctPlainText = "text/plain; charset=utf-8"
	ctSVG       = "image/svg+xml"
)

var mapEncoding2CT = map[api.EncodingEnum]string{
	api.EncoderHTML:   ctHTML,
	api.EncoderNative: ctPlainText,
	api.EncoderZJSON:  ctJSON,
	api.EncoderText:   ctPlainText,
	api.EncoderZmk:    ctPlainText,
}

func encoding2ContentType(enc api.EncodingEnum) string {
	if ct, ok := mapEncoding2CT[enc]; ok {
		return ct
	}
	return "application/octet-stream"
}

var mapSyntax2CT = map[string]string{
	"css":               "text/css; charset=utf-8",
	api.ValueSyntaxDraw: ctSVG,
	api.ValueSyntaxGif:  "image/gif",
	api.ValueSyntaxHTML: "text/html; charset=utf-8",
	"jpeg":              "image/jpeg",
	"jpg":               "image/jpeg",
	"js":                "text/javascript; charset=utf-8",
	"pdf":               "application/pdf",
	"png":               "image/png",
	api.ValueSyntaxSVG:  ctSVG,
	"xml":               "text/xml; charset=utf-8",
	api.ValueSyntaxZmk:  "text/x-zmk; charset=utf-8",
	"plain":             ctPlainText,
	api.ValueSyntaxText: ctPlainText,
	"markdown":          "text/markdown; charset=utf-8",
	"md":                "text/markdown; charset=utf-8",
	"mustache":          ctPlainText,
}

func syntax2contentType(syntax string) (string, bool) {
	contentType, ok := mapSyntax2CT[syntax]
	return contentType, ok
}
