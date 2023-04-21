//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package parser_test

import (
	"testing"

	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/strfun"

	_ "zettelstore.de/z/parser/blob"       // Allow to use BLOB parser.
	_ "zettelstore.de/z/parser/draw"       // Allow to use draw parser.
	_ "zettelstore.de/z/parser/markdown"   // Allow to use markdown parser.
	_ "zettelstore.de/z/parser/none"       // Allow to use none parser.
	_ "zettelstore.de/z/parser/plain"      // Allow to use plain parser.
	_ "zettelstore.de/z/parser/zettelmark" // Allow to use zettelmark parser.
)

func TestParserType(t *testing.T) {
	syntaxSet := strfun.NewSet(parser.GetSyntaxes()...)
	testCases := []struct {
		syntax string
		ast    bool
		text   bool
		image  bool
	}{
		{meta.SyntaxHTML, false, true, false},
		{meta.SyntaxCSS, false, true, false},
		{meta.SyntaxDraw, true, true, false},
		{meta.SyntaxGif, false, false, true},
		{meta.SyntaxJPEG, false, false, true},
		{meta.SyntaxJPG, false, false, true},
		{meta.SyntaxMarkdown, true, true, false},
		{meta.SyntaxMD, true, true, false},
		{meta.SyntaxMustache, false, true, false},
		{meta.SyntaxNone, false, false, false},
		{meta.SyntaxPlain, false, true, false},
		{meta.SyntaxPNG, false, false, true},
		{meta.SyntaxSVG, false, true, true},
		{meta.SyntaxSxn, false, true, false},
		{meta.SyntaxText, false, true, false},
		{meta.SyntaxTxt, false, true, false},
		{meta.SyntaxWebp, false, false, true},
		{meta.SyntaxZmk, true, true, false},
	}
	for _, tc := range testCases {
		delete(syntaxSet, tc.syntax)
		if got := parser.IsASTParser(tc.syntax); got != tc.ast {
			t.Errorf("Syntax %q is AST: %v, but got %v", tc.syntax, tc.ast, got)
		}
		if got := parser.IsTextFormat(tc.syntax); got != tc.text {
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
