//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package content manages content handling within the web package.
// It translates syntax values into content types, and vice versa.
package content

import "zettelstore.de/c/api"

const (
	UnknownMIME  = "application/octet-stream"
	mimeHTML     = "text/html; charset=utf-8"
	mimeJPEG     = "image/jpeg"
	mimeMarkdown = "text/markdown; charset=utf-8"
	JSON         = "application/json"
	PlainText    = "text/plain; charset=utf-8"
)

var encoding2mime = map[api.EncodingEnum]string{
	api.EncoderHTML:  mimeHTML,
	api.EncoderMD:    mimeMarkdown,
	api.EncoderSexpr: PlainText,
	api.EncoderText:  PlainText,
	api.EncoderZJSON: JSON,
	api.EncoderZmk:   PlainText,
}

// MIMEFromEncoding returns the MIME encoding for a given zettel encoding
func MIMEFromEncoding(enc api.EncodingEnum) string {
	if m, found := encoding2mime[enc]; found {
		return m
	}
	return UnknownMIME
}

var syntax2mime = map[string][]string{
	api.ValueSyntaxCSS:      {"text/css; charset=utf-8"},
	"draw":                  {PlainText},
	api.ValueSyntaxGif:      {"image/gif"},
	api.ValueSyntaxHTML:     {mimeHTML},
	"jpeg":                  {mimeJPEG},
	"jpg":                   {mimeJPEG},
	api.ValueSyntaxMarkdown: {mimeMarkdown},
	api.ValueSyntaxMD:       {mimeMarkdown},
	api.ValueSyntaxMustache: {PlainText},
	"none":                  {""},
	"plain":                 {PlainText},
	"png":                   {"image/png"},
	api.ValueSyntaxSVG:      {"image/svg+xml"},
	"txt":                   {PlainText},
	api.ValueSyntaxText:     {PlainText},
	"webp":                  {"image/webp"},
	api.ValueSyntaxZmk:      {"text/x-zmk; charset=utf-8"},

	// Additional syntaxes that are parsed as plain
	"js":  {"text/javascript; charset=utf-8"},
	"pdf": {"application/pdf"},
	"xml": {"text/xml; charset=utf-8"},
}

// MIMEFromSyntax returns a MIME encoding for a given syntax value.
func MIMEFromSyntax(syntax string) string {
	if ms, found := syntax2mime[syntax]; found && len(ms) > 0 {
		return ms[0]
	}
	return UnknownMIME
}
