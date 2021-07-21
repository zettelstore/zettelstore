//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package tests provides some higher-level tests.
package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"zettelstore.de/z/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/encoder"
	_ "zettelstore.de/z/encoder/htmlenc"
	_ "zettelstore.de/z/encoder/jsonenc"
	_ "zettelstore.de/z/encoder/nativeenc"
	_ "zettelstore.de/z/encoder/textenc"
	_ "zettelstore.de/z/encoder/zmkenc"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
	_ "zettelstore.de/z/parser/markdown"
	_ "zettelstore.de/z/parser/zettelmark"
)

type markdownTestCase struct {
	Markdown  string `json:"markdown"`
	HTML      string `json:"html"`
	Example   int    `json:"example"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Section   string `json:"section"`
}

// exceptions lists all CommonMark tests that should not be tested for identical HTML output
var exceptions = []string{
	" - foo\n   - bar\n\t - baz\n", // 9
	"<script type=\"text/javascript\">\n// JavaScript example\n\ndocument.getElementById(\"demo\").innerHTML = \"Hello JavaScript!\";\n</script>\nokay\n", // 170
	"<script>\nfoo\n</script>1. *bar*\n",                       // 178
	"- foo\n  - bar\n    - baz\n      - boo\n",                 // 294
	"10) foo\n    - bar\n",                                     // 296
	"- # Foo\n- Bar\n  ---\n  baz\n",                           // 300
	"- foo\n\n- bar\n\n\n- baz\n",                              // 306
	"- foo\n  - bar\n    - baz\n\n\n      bim\n",               // 307
	"1. a\n\n  2. b\n\n   3. c\n",                              // 311
	"1. a\n\n  2. b\n\n    3. c\n",                             // 313
	"- a\n- b\n\n- c\n",                                        // 314
	"* a\n*\n\n* c\n",                                          // 315
	"- a\n- b\n\n  [ref]: /url\n- d\n",                         // 317
	"- a\n  - b\n\n    c\n- d\n",                               // 319
	"* a\n  > b\n  >\n* c\n",                                   // 320
	"- a\n  > b\n  ```\n  c\n  ```\n- d\n",                     // 321
	"- a\n  - b\n",                                             // 323
	"<http://foo.bar.`baz>`\n",                                 // 345
	"[foo<http://example.com/?search=](uri)>\n",                // 525
	"[foo<http://example.com/?search=][ref]>\n\n[ref]: /uri\n", // 537
	"<http://example.com?find=\\*>\n",                          // 581
	"<http://foo.bar.baz/test?q=hello&id=22&boolean>\n",        // 594
}

var reHeadingID = regexp.MustCompile(` id="[^"]*"`)

func TestEncoderAvailability(t *testing.T) {
	t.Parallel()
	encoderMissing := false
	for _, format := range formats {
		enc := encoder.Create(format, nil)
		if enc == nil {
			t.Errorf("No encoder for %q found", format)
			encoderMissing = true
		}
	}
	if encoderMissing {
		panic("At least one encoder is missing. See test log")
	}
}

func TestMarkdownSpec(t *testing.T) {
	t.Parallel()
	content, err := os.ReadFile("../testdata/markdown/spec.json")
	if err != nil {
		panic(err)
	}
	var testcases []markdownTestCase
	if err = json.Unmarshal(content, &testcases); err != nil {
		panic(err)
	}
	excMap := make(map[string]bool, len(exceptions))
	for _, exc := range exceptions {
		excMap[exc] = true
	}
	for _, tc := range testcases {
		ast := parser.ParseBlocks(input.NewInput(tc.Markdown), nil, "markdown")
		testAllEncodings(t, tc, ast)
		if _, found := excMap[tc.Markdown]; !found {
			testHTMLEncoding(t, tc, ast)
		}
		testZmkEncoding(t, tc, ast)
	}
}

func testAllEncodings(t *testing.T, tc markdownTestCase, ast ast.BlockSlice) {
	var sb strings.Builder
	testID := tc.Example*100 + 1
	for _, format := range formats {
		t.Run(fmt.Sprintf("Encode %v %v", format, testID), func(st *testing.T) {
			encoder.Create(format, nil).WriteBlocks(&sb, ast)
			sb.Reset()
		})
	}
}

func testHTMLEncoding(t *testing.T, tc markdownTestCase, ast ast.BlockSlice) {
	htmlEncoder := encoder.Create(api.EncoderHTML, &encoder.Environment{Xhtml: true})
	var sb strings.Builder
	testID := tc.Example*100 + 1
	t.Run(fmt.Sprintf("Encode md html %v", testID), func(st *testing.T) {
		htmlEncoder.WriteBlocks(&sb, ast)
		gotHTML := sb.String()
		sb.Reset()

		mdHTML := tc.HTML
		mdHTML = strings.ReplaceAll(mdHTML, "\"MAILTO:", "\"mailto:")
		gotHTML = strings.ReplaceAll(gotHTML, " class=\"zs-external\"", "")
		gotHTML = strings.ReplaceAll(gotHTML, "%2A", "*") // url.QueryEscape
		if strings.Count(gotHTML, "<h") > 0 {
			gotHTML = reHeadingID.ReplaceAllString(gotHTML, "")
		}
		if gotHTML != mdHTML {
			mdHTML = strings.ReplaceAll(mdHTML, "<li>\n", "<li>")
			if gotHTML != mdHTML {
				st.Errorf("\nCMD: %q\nExp: %q\nGot: %q", tc.Markdown, mdHTML, gotHTML)
			}
		}
	})
}

func testZmkEncoding(t *testing.T, tc markdownTestCase, ast ast.BlockSlice) {
	zmkEncoder := encoder.Create(api.EncoderZmk, nil)
	var sb strings.Builder
	testID := tc.Example*100 + 1
	t.Run(fmt.Sprintf("Encode zmk %14d", testID), func(st *testing.T) {
		zmkEncoder.WriteBlocks(&sb, ast)
		gotFirst := sb.String()
		sb.Reset()

		testID = tc.Example*100 + 2
		secondAst := parser.ParseBlocks(input.NewInput(gotFirst), nil, "zmk")
		zmkEncoder.WriteBlocks(&sb, secondAst)
		gotSecond := sb.String()
		sb.Reset()

		// if gotFirst != gotSecond {
		// 	st.Errorf("\nCMD: %q\n1st: %q\n2nd: %q", tc.Markdown, gotFirst, gotSecond)
		// }

		testID = tc.Example*100 + 3
		thirdAst := parser.ParseBlocks(input.NewInput(gotFirst), nil, "zmk")
		zmkEncoder.WriteBlocks(&sb, thirdAst)
		gotThird := sb.String()
		sb.Reset()

		if gotSecond != gotThird {
			st.Errorf("\n1st: %q\n2nd: %q", gotSecond, gotThird)
		}
	})

}
