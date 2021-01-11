//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
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
	"io/ioutil"
	"regexp"
	"strings"
	"testing"

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
	" - foo\n   - bar\n\t - baz\n",                             // 9
	"- foo\n  - bar\n    - baz\n      - boo\n",                 // 264
	"10) foo\n    - bar\n",                                     // 266
	"- # Foo\n- Bar\n  ---\n  baz\n",                           // 270
	"- foo\n\n- bar\n\n\n- baz\n",                              // 276
	"- foo\n  - bar\n    - baz\n\n\n      bim\n",               // 277
	"1. a\n\n  2. b\n\n   3. c\n",                              // 281
	"1. a\n\n  2. b\n\n    3. c\n",                             // 283
	"- a\n- b\n\n- c\n",                                        // 284
	"* a\n*\n\n* c\n",                                          // 285
	"- a\n- b\n\n  [ref]: /url\n- d\n",                         // 287
	"- a\n  - b\n\n    c\n- d\n",                               // 289
	"* a\n  > b\n  >\n* c\n",                                   // 290
	"- a\n  > b\n  ```\n  c\n  ```\n- d\n",                     // 291
	"- a\n  - b\n",                                             // 293
	"<http://example.com?find=\\*>\n",                          // 306
	"<http://foo.bar.`baz>`\n",                                 // 346
	"[foo<http://example.com/?search=](uri)>\n",                // 522
	"[foo<http://example.com/?search=][ref]>\n\n[ref]: /uri\n", // 534
	"<http://foo.bar.baz/test?q=hello&id=22&boolean>\n",        // 591
}

var reHeadingID = regexp.MustCompile(` id="[^"]*"`)

func TestMarkdownSpec(t *testing.T) {
	content, err := ioutil.ReadFile("../testdata/markdown/spec.json")
	if err != nil {
		panic(err)
	}
	var testcases []markdownTestCase
	if err = json.Unmarshal(content, &testcases); err != nil {
		panic(err)
	}
	for _, format := range formats {
		enc := encoder.Create(format)
		if enc == nil {
			panic(fmt.Sprintf("No encoder for %q found", format))
		}
	}
	excMap := make(map[string]bool, len(exceptions))
	for _, exc := range exceptions {
		excMap[exc] = true
	}
	htmlEncoder := encoder.Create("html", &encoder.BoolOption{Key: "xhtml", Value: true})
	zmkEncoder := encoder.Create("zmk")
	var sb strings.Builder
	for _, tc := range testcases {
		testID := tc.Example*100 + 1
		ast := parser.ParseBlocks(input.NewInput(tc.Markdown), nil, "markdown")

		for _, format := range formats {
			t.Run(fmt.Sprintf("Encode %v %v", format, testID), func(st *testing.T) {
				encoder.Create(format).WriteBlocks(&sb, ast)
				sb.Reset()
			})
		}
		if _, found := excMap[tc.Markdown]; !found {
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
					mdHTML := strings.ReplaceAll(mdHTML, "<li>\n", "<li>")
					if gotHTML != mdHTML {
						st.Errorf("\nCMD: %q\nExp: %q\nGot: %q", tc.Markdown, mdHTML, gotHTML)
					}
				}
			})
		}
		t.Run(fmt.Sprintf("Encode zmk %14d", testID), func(st *testing.T) {
			zmkEncoder.WriteBlocks(&sb, ast)
			gotFirst := sb.String()
			sb.Reset()

			testID = tc.Example*100 + 2
			secondAst := parser.ParseBlocks(input.NewInput(gotFirst), nil, "zmk")
			zmkEncoder.WriteBlocks(&sb, secondAst)
			gotSecond := sb.String()
			sb.Reset()

			if gotFirst != gotSecond {
				//st.Errorf("\nCMD: %q\n1st: %q\n2nd: %q", tc.Markdown, gotFirst, gotSecond)
			}

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
}
