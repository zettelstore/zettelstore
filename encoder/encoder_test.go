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
	descr  string
	zmk    string
	inline bool
	expect expectMap
}

type expectMap map[api.EncodingEnum]string

const useZmk = "\000"

var testCases = []testCase{
	{
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
		descr:  "Insert formatting",
		zmk:    "__insert__{-}",
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Insert","i":[{"t":"Text","s":"insert"}]}]`,
			api.EncoderHTML:   "<ins>insert</ins>",
			api.EncoderNative: `Insert [Text "insert"]`,
			api.EncoderText:   "insert",
			api.EncoderZmk:    useZmk,
		},
	},
	{
		descr:  "Strike formatting",
		zmk:    "~~strike~~",
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Strikethrough","i":[{"t":"Text","s":"strike"}]}]`,
			api.EncoderHTML:   "<s>strike</s>",
			api.EncoderNative: `Strikethrough [Text "strike"]`,
			api.EncoderText:   "strike",
			api.EncoderZmk:    useZmk,
		},
	},
	{
		descr:  "Delete formatting",
		zmk:    "~~delete~~{-}",
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Delete","i":[{"t":"Text","s":"delete"}]}]`,
			api.EncoderHTML:   "<del>delete</del>",
			api.EncoderNative: `Delete [Text "delete"]`,
			api.EncoderText:   "delete",
			api.EncoderZmk:    useZmk,
		},
	},
	{
		descr:  "Update formatting",
		zmk:    "~~old~~{-}__new__{-}",
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Delete","i":[{"t":"Text","s":"old"}]},{"t":"Insert","i":[{"t":"Text","s":"new"}]}]`,
			api.EncoderHTML:   "<del>old</del><ins>new</ins>",
			api.EncoderNative: `Delete [Text "old"],Insert [Text "new"]`,
			api.EncoderText:   "oldnew",
			api.EncoderZmk:    useZmk,
		},
	},
	{
		descr:  "Monospace formatting",
		zmk:    "''monospace''",
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Mono","i":[{"t":"Text","s":"monospace"}]}]`,
			api.EncoderHTML:   `<span style="font-family:monospace">monospace</span>`,
			api.EncoderNative: `Mono [Text "monospace"]`,
			api.EncoderText:   "monospace",
			api.EncoderZmk:    useZmk,
		},
	},
	{
		descr:  "Superscript formatting",
		zmk:    "^^superscript^^",
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Super","i":[{"t":"Text","s":"superscript"}]}]`,
			api.EncoderHTML:   `<sup>superscript</sup>`,
			api.EncoderNative: `Super [Text "superscript"]`,
			api.EncoderText:   `superscript`,
			api.EncoderZmk:    useZmk,
		},
	},
	{
		descr:  "Subscript formatting",
		zmk:    ",,subscript,,",
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Sub","i":[{"t":"Text","s":"subscript"}]}]`,
			api.EncoderHTML:   `<sub>subscript</sub>`,
			api.EncoderNative: `Sub [Text "subscript"]`,
			api.EncoderText:   `subscript`,
			api.EncoderZmk:    useZmk,
		},
	},
	{
		descr:  "Quotes formatting",
		zmk:    `""quotes""`,
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Quote","i":[{"t":"Text","s":"quotes"}]}]`,
			api.EncoderHTML:   `"quotes"`,
			api.EncoderNative: `Quote [Text "quotes"]`,
			api.EncoderText:   `quotes`,
			api.EncoderZmk:    useZmk,
		},
	},
	{
		descr:  "Quotes formatting (german)",
		zmk:    `""quotes""{lang=de}`,
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Quote","a":{"lang":"de"},"i":[{"t":"Text","s":"quotes"}]}]`,
			api.EncoderHTML:   `<span lang="de">&bdquo;quotes&ldquo;</span>`,
			api.EncoderNative: `Quote ("",[lang="de"]) [Text "quotes"]`,
			api.EncoderText:   `quotes`,
			api.EncoderZmk:    `""quotes""{lang="de"}`,
		},
	},
	{
		descr:  "Quotation formatting",
		zmk:    `<<quotation<<`,
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Quotation","i":[{"t":"Text","s":"quotation"}]}]`,
			api.EncoderHTML:   `<q>quotation</q>`,
			api.EncoderNative: `Quotation [Text "quotation"]`,
			api.EncoderText:   `quotation`,
			api.EncoderZmk:    useZmk,
		},
	},
	{
		descr:  "Small formatting",
		zmk:    `;;small;;`,
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Small","i":[{"t":"Text","s":"small"}]}]`,
			api.EncoderHTML:   `<small>small</small>`,
			api.EncoderNative: `Small [Text "small"]`,
			api.EncoderText:   `small`,
			api.EncoderZmk:    useZmk,
		},
	},
	{
		descr:  "Span formatting",
		zmk:    `::span::`,
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Span","i":[{"t":"Text","s":"span"}]}]`,
			api.EncoderHTML:   `<span>span</span>`,
			api.EncoderNative: `Span [Text "span"]`,
			api.EncoderText:   `span`,
			api.EncoderZmk:    useZmk,
		},
	},
	{
		descr:  "Code formatting",
		zmk:    "``code``",
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Code","s":"code"}]`,
			api.EncoderHTML:   `<code>code</code>`,
			api.EncoderNative: `Code "code"`,
			api.EncoderText:   `code`,
			api.EncoderZmk:    useZmk,
		},
	},
	{
		descr:  "Input formatting",
		zmk:    `++input++`,
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Input","s":"input"}]`,
			api.EncoderHTML:   `<kbd>input</kbd>`,
			api.EncoderNative: `Input "input"`,
			api.EncoderText:   `input`,
			api.EncoderZmk:    useZmk,
		},
	},
	{
		descr:  "Output formatting",
		zmk:    `==output==`,
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Output","s":"output"}]`,
			api.EncoderHTML:   `<samp>output</samp>`,
			api.EncoderNative: `Output "output"`,
			api.EncoderText:   `output`,
			api.EncoderZmk:    useZmk,
		},
	},
	{
		descr:  "Nested Span Quote formatting",
		zmk:    `::""abc""::{lang=fr}`,
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Span","a":{"lang":"fr"},"i":[{"t":"Quote","i":[{"t":"Text","s":"abc"}]}]}]`,
			api.EncoderHTML:   `<span lang="fr">&laquo;&nbsp;abc&nbsp;&raquo;</span>`,
			api.EncoderNative: `Span ("",[lang="fr"]) [Quote [Text "abc"]]`,
			api.EncoderText:   `abc`,
			api.EncoderZmk:    `::""abc""::{lang="fr"}`,
		},
	},
	{
		descr:  "Simple Citation",
		zmk:    `[@Stern18]`,
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Cite","s":"Stern18"}]`,
			api.EncoderHTML:   `Stern18`, // TODO
			api.EncoderNative: `Cite "Stern18"`,
			api.EncoderText:   ``,
			api.EncoderZmk:    useZmk,
		},
	},
	{
		descr:  "No comment",
		zmk:    `% comment`,
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Text","s":"%"},{"t":"Space"},{"t":"Text","s":"comment"}]`,
			api.EncoderHTML:   `% comment`,
			api.EncoderNative: `Text "%",Space,Text "comment"`,
			api.EncoderText:   `% comment`,
			api.EncoderZmk:    useZmk,
		},
	},
	{
		descr:  "Line comment",
		zmk:    `%% line comment`,
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Comment","s":"line comment"}]`,
			api.EncoderHTML:   `<!-- line comment -->`,
			api.EncoderNative: `Comment "line comment"`,
			api.EncoderText:   ``,
			api.EncoderZmk:    useZmk,
		},
	},
	{
		descr:  "Comment after text",
		zmk:    `Text %% comment`,
		inline: true,
		expect: expectMap{
			// TODO: Space before comment should be removed
			api.EncoderDJSON: `[{"t":"Text","s":"Text"},{"t":"Space"},{"t":"Comment","s":"comment"}]`,
			api.EncoderHTML:  `Text <!-- comment -->`,
			// TODO: Space before comment should be removed
			api.EncoderNative: `Text "Text",Space,Comment "comment"`,
			// TODO: Space at end should be removed
			api.EncoderText: `Text `,
			api.EncoderZmk:  useZmk,
		},
	},
	{
		descr:  "Simple footnote",
		zmk:    `[^footnote]`,
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Footnote","i":[{"t":"Text","s":"footnote"}]}]`,
			api.EncoderHTML:   `<sup id="fnref:0"><a href="#fn:0" class="zs-footnote-ref" role="doc-noteref">0</a></sup>`,
			api.EncoderNative: `Footnote [Text "footnote"]`,
			api.EncoderText:   ` footnote`, // TODO: remove leading space
			api.EncoderZmk:    useZmk,
		},
	},
	{
		descr:  "Simple mark",
		zmk:    `[!mark]`,
		inline: true,
		expect: expectMap{
			api.EncoderDJSON:  `[{"t":"Mark","s":"mark"}]`,
			api.EncoderHTML:   ``,
			api.EncoderNative: `Mark "mark"`,
			api.EncoderText:   ``,
			api.EncoderZmk:    useZmk,
		},
	},
	{
		descr:  "",
		zmk:    ``,
		inline: false,
		expect: expectMap{
			api.EncoderDJSON:  `[]`,
			api.EncoderHTML:   ``,
			api.EncoderNative: ``,
			api.EncoderText:   ``,
			api.EncoderZmk:    useZmk,
		},
	},
}

func TestEncoder(t *testing.T) {
	for testNum, tc := range testCases {
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
