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
	"strings"
	"testing"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/encoder"
	_ "zettelstore.de/z/encoder/djsonenc"
	_ "zettelstore.de/z/encoder/htmlenc"
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

func TestEncoderAvailability(t *testing.T) {
	t.Parallel()
	encoderMissing := false
	for _, enc := range encodings {
		enc := encoder.Create(enc, nil)
		if enc == nil {
			t.Errorf("No encoder for %q found", enc)
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

	for _, tc := range testcases {
		ast := parser.ParseBlocks(input.NewInput(tc.Markdown), nil, "markdown")
		testAllEncodings(t, tc, ast)
		testZmkEncoding(t, tc, ast)
	}
}

func testAllEncodings(t *testing.T, tc markdownTestCase, ast *ast.BlockListNode) {
	var sb strings.Builder
	testID := tc.Example*100 + 1
	for _, enc := range encodings {
		t.Run(fmt.Sprintf("Encode %v %v", enc, testID), func(st *testing.T) {
			encoder.Create(enc, nil).WriteBlocks(&sb, ast)
			sb.Reset()
		})
	}
}

func testZmkEncoding(t *testing.T, tc markdownTestCase, ast *ast.BlockListNode) {
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
