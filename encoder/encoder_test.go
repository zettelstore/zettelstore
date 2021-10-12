//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package encoder_test

import (
	"fmt"
	"strings"
	"testing"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"

	_ "zettelstore.de/z/encoder/djsonenc"  // Allow to use DJSON encoder.
	_ "zettelstore.de/z/encoder/htmlenc"   // Allow to use HTML encoder.
	_ "zettelstore.de/z/encoder/nativeenc" // Allow to use native encoder.
	_ "zettelstore.de/z/encoder/textenc"   // Allow to use text encoder.
	_ "zettelstore.de/z/encoder/zmkenc"    // Allow to use zmk encoder.
	_ "zettelstore.de/z/parser/zettelmark" // Allow to use zettelmark parser.
)

type testCase struct {
	num    int
	descr  string
	zmk    string
	inline bool
	expect expectMap
}

type expectMap map[api.EncodingEnum]string

const useZmk = "\000"

var testCases = []testCase{
	{
		num:   0,
		descr: "Empty Zettelmarkup should produce near nothing",
		zmk:   "",
		expect: expectMap{
			api.EncoderDJSON:  `[]`,
			api.EncoderHTML:   "",
			api.EncoderNative: ``,
			api.EncoderText:   "",
			api.EncoderZmk:    useZmk,
		},
	},
	{
		num:    1,
		descr:  "Empty Zettelmarkup should produce near nothing (inline)",
		zmk:    "",
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[]`,
			api.EncoderHTML:   "",
			api.EncoderNative: ``,
			api.EncoderText:   "",
			api.EncoderZmk:    useZmk,
		},
	},
	{
		num:   2,
		descr: "Simple text: Hello, world",
		zmk:   "Hello, world",
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Para","i":[{"t":"Text","s":"Hello,"},{"t":"Space"},{"t":"Text","s":"world"}]}]`,
			api.EncoderHTML:   "<p>Hello, world</p>\n", // TODO: remove \n
			api.EncoderNative: `[Para Text "Hello,",Space,Text "world"]`,
			api.EncoderText:   "Hello, world",
			api.EncoderZmk:    "Hello, world\n\n", // TODO: remove \n
		},
	},
	{
		num:    3,
		descr:  "Simple text: Hello, world (inline)",
		zmk:    "Hello, world",
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Text","s":"Hello,"},{"t":"Space"},{"t":"Text","s":"world"}]`,
			api.EncoderHTML:   "Hello, world",
			api.EncoderNative: `Text "Hello,",Space,Text "world"`,
			api.EncoderText:   "Hello, world",
			api.EncoderZmk:    useZmk,
		},
	},
	{
		num:    10,
		descr:  "Italic formatting",
		zmk:    "//italic//",
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Italic","i":[{"t":"Text","s":"italic"}]}]`,
			api.EncoderHTML:   "<i>italic</i>",
			api.EncoderNative: `Italic [Text "italic"]`,
			api.EncoderText:   "italic",
			api.EncoderZmk:    useZmk,
		},
	},
	{
		num:    11,
		descr:  "Emphasized formatting",
		zmk:    "//emph//{-}",
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Emph","i":[{"t":"Text","s":"emph"}]}]`,
			api.EncoderHTML:   "<em>emph</em>",
			api.EncoderNative: `Emph [Text "emph"]`,
			api.EncoderText:   "emph",
			api.EncoderZmk:    useZmk,
		},
	},
	{
		num:    12,
		descr:  "Bold formatting",
		zmk:    "**bold**",
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Bold","i":[{"t":"Text","s":"bold"}]}]`,
			api.EncoderHTML:   "<b>bold</b>",
			api.EncoderNative: `Bold [Text "bold"]`,
			api.EncoderText:   "bold",
			api.EncoderZmk:    useZmk,
		},
	},
	{
		num:    13,
		descr:  "Strong formatting",
		zmk:    "**strong**{-}",
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Strong","i":[{"t":"Text","s":"strong"}]}]`,
			api.EncoderHTML:   "<strong>strong</strong>",
			api.EncoderNative: `Strong [Text "strong"]`,
			api.EncoderText:   "strong",
			api.EncoderZmk:    useZmk,
		},
	},
	{
		num:    14,
		descr:  "Underline formatting",
		zmk:    "__underline__",
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Underline","i":[{"t":"Text","s":"underline"}]}]`,
			api.EncoderHTML:   "<u>underline</u>",
			api.EncoderNative: `Underline [Text "underline"]`,
			api.EncoderText:   "underline",
			api.EncoderZmk:    useZmk,
		},
	},
	{
		num:    0,
		descr:  "",
		zmk:    "",
		inline: false,
		expect: expectMap{
			api.EncoderDJSON:  `[]`,
			api.EncoderHTML:   "",
			api.EncoderNative: ``,
			api.EncoderText:   "",
			api.EncoderZmk:    useZmk,
		},
	},
}

func TestEncoder(t *testing.T) {
	for i, tc := range testCases {
		inp := input.NewInput(tc.zmk)
		var pe parserEncoder
		if tc.inline {
			pe = &peInlines{iln: parser.ParseInlines(inp, api.ValueSyntaxZmk)}
		} else {
			pe = &peBlocks{bln: parser.ParseBlocks(inp, nil, api.ValueSyntaxZmk)}
		}
		for enc, exp := range tc.expect {
			encdr := encoder.Create(enc, nil)
			got, err := pe.encode(encdr)
			if err != nil {
				t.Error(err)
				continue
			}
			if enc == api.EncoderZmk && exp == "\000" {
				exp = tc.zmk
			}
			if got != exp {
				testNum := tc.num
				if testNum <= 0 {
					testNum = i
				}
				prefix := fmt.Sprintf("Test #%d", testNum)
				if d := tc.descr; d != "" {
					prefix += "\nReason:   " + d
				}
				if tc.inline {
					prefix += "\nMode:     inline"
				} else {
					prefix += "\nMode:     block"
				}
				t.Errorf("%s\nEncoder:  %s\nExpected: %q\nGot:      %q", prefix, enc, exp, got)
			}
		}
	}
}

type parserEncoder interface {
	encode(encoder.Encoder) (string, error)
}

type peInlines struct {
	iln *ast.InlineListNode
}

func (in peInlines) encode(encdr encoder.Encoder) (string, error) {
	var sb strings.Builder
	if _, err := encdr.WriteInlines(&sb, in.iln); err != nil {
		return "", err
	}
	return sb.String(), nil
}

type peBlocks struct {
	bln *ast.BlockListNode
}

func (bl peBlocks) encode(encdr encoder.Encoder) (string, error) {
	var sb strings.Builder
	if _, err := encdr.WriteBlocks(&sb, bl.bln); err != nil {
		return "", err
	}
	return sb.String(), nil

}
