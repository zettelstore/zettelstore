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
	ctMarkdown  = "text/markdown; charset=utf-8"
	ctPlainText = "text/plain; charset=utf-8"
	ctSVG       = "image/svg+xml"
)

var mapEncoding2CT = map[api.EncodingEnum]string{
	api.EncoderHTML:  ctHTML,
	api.EncoderMD:    ctMarkdown,
	api.EncoderSexpr: ctPlainText,
	api.EncoderText:  ctPlainText,
	api.EncoderZJSON: ctJSON,
	api.EncoderZmk:   ctPlainText,
}

func encoding2ContentType(enc api.EncodingEnum) string {
	if ct, ok := mapEncoding2CT[enc]; ok {
		return ct
	}
	return "application/octet-stream"
}

var mapSyntax2CT = map[string]string{
	api.ValueSyntaxCSS:      "text/css; charset=utf-8",
	api.ValueSyntaxGif:      "image/gif",
	api.ValueSyntaxHTML:     "text/html; charset=utf-8",
	"jpeg":                  "image/jpeg",
	"jpg":                   "image/jpeg",
	"js":                    "text/javascript; charset=utf-8",
	"pdf":                   "application/pdf",
	"png":                   "image/png",
	api.ValueSyntaxSVG:      ctSVG,
	"xml":                   "text/xml; charset=utf-8",
	api.ValueSyntaxZmk:      "text/x-zmk; charset=utf-8",
	"plain":                 ctPlainText,
	api.ValueSyntaxText:     ctPlainText,
	api.ValueSyntaxMarkdown: ctMarkdown,
	api.ValueSyntaxMD:       ctMarkdown,
	api.ValueSyntaxMustache: ctPlainText,
}

func syntax2contentType(syntax string) (string, bool) {
	contentType, ok := mapSyntax2CT[syntax]
	return contentType, ok
}
