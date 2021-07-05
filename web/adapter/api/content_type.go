//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package api provides api handlers for web requests.
package api

import "zettelstore.de/z/encoder"

const plainText = "text/plain; charset=utf-8"

var mapFormat2CT = map[encoder.Enum]string{
	encoder.EncoderHTML:   "text/html; charset=utf-8",
	encoder.EncoderNative: plainText,
	encoder.EncoderJSON:   "application/json",
	encoder.EncoderDJSON:  "application/json",
	encoder.EncoderText:   plainText,
	encoder.EncoderZmk:    plainText,
	encoder.EncoderRaw:    plainText, // In some cases...
}

func format2ContentType(format encoder.Enum) string {
	ct, ok := mapFormat2CT[format]
	if !ok {
		return "application/octet-stream"
	}
	return ct
}

var mapSyntax2CT = map[string]string{
	"css":      "text/css; charset=utf-8",
	"gif":      "image/gif",
	"html":     "text/html; charset=utf-8",
	"jpeg":     "image/jpeg",
	"jpg":      "image/jpeg",
	"js":       "text/javascript; charset=utf-8",
	"pdf":      "application/pdf",
	"png":      "image/png",
	"svg":      "image/svg+xml",
	"xml":      "text/xml; charset=utf-8",
	"zmk":      "text/x-zmk; charset=utf-8",
	"plain":    plainText,
	"text":     plainText,
	"markdown": "text/markdown; charset=utf-8",
	"md":       "text/markdown; charset=utf-8",
	"mustache": plainText,
	//"graphviz":      "text/vnd.graphviz; charset=utf-8",
}

func syntax2contentType(syntax string) (string, bool) {
	contentType, ok := mapSyntax2CT[syntax]
	return contentType, ok
}
