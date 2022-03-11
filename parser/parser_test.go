//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package parser provides a generic interface to a range of different parsers.
package parser_test

import (
	"testing"

	"zettelstore.de/c/api"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/strfun"

	_ "zettelstore.de/z/parser/blob"       // Allow to use BLOB parser.
	_ "zettelstore.de/z/parser/markdown"   // Allow to use markdown parser.
	_ "zettelstore.de/z/parser/none"       // Allow to use none parser.
	_ "zettelstore.de/z/parser/plain"      // Allow to use plain parser.
	_ "zettelstore.de/z/parser/zettelmark" // Allow to use zettelmark parser.
)

func TestParserType(t *testing.T) {
	syntaxSet := strfun.NewSet(parser.GetSyntaxes()...)
	testCases := []struct {
		syntax string
		text   bool
		image  bool
	}{
		{api.ValueSyntaxHTML, false, false},
		{"css", false, false},
		{api.ValueSyntaxGif, false, true},
		{"jpeg", false, true},
		{"jpg", false, true},
		{"markdown", true, false},
		{"md", true, false},
		{"mustache", false, false},
		{api.ValueSyntaxNone, false, false},
		{"plain", false, false},
		{"png", false, true},
		{api.ValueSyntaxSVG, false, true},
		{api.ValueSyntaxText, false, false},
		{"txt", false, false},
		{api.ValueSyntaxZmk, true, false},
	}
	for _, tc := range testCases {
		delete(syntaxSet, tc.syntax)
		if got := parser.IsTextParser(tc.syntax); got != tc.text {
			t.Errorf("Syntax %q is text: %v, but got %v", tc.syntax, tc.text, got)
		}
		if got := parser.IsImageFormat(tc.syntax); got != tc.image {
			t.Errorf("Syntax %q is image: %v, but got %v", tc.syntax, tc.image, got)
		}
	}
	for syntax := range syntaxSet {
		t.Errorf("Forgot to test syntax %q", syntax)
	}
}
