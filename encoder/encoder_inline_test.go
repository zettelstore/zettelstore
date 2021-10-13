//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package encoder_test

var tcsInline = []testCase{
	{
		descr: "Empty Zettelmarkup should produce near nothing",
		zmk:   "",
		expect: expectMap{
			encoderDJSON:  `[]`,
			encoderHTML:   "",
			encoderNative: ``,
			encoderText:   "",
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Empty Zettelmarkup should produce near nothing (inline)",
		zmk:    "",
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[]`,
			encoderHTML:   "",
			encoderNative: ``,
			encoderText:   "",
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Simple text: Hello, world",
		zmk:   "Hello, world",
		expect: expectMap{
			encoderDJSON:  `[{"t":"Para","i":[{"t":"Text","s":"Hello,"},{"t":"Space"},{"t":"Text","s":"world"}]}]`,
			encoderHTML:   "<p>Hello, world</p>\n", // TODO: remove \n
			encoderNative: `[Para Text "Hello,",Space,Text "world"]`,
			encoderText:   "Hello, world",
			encoderZmk:    "Hello, world\n\n", // TODO: remove \n
		},
	},
	{
		descr:  "Simple text: Hello, world (inline)",
		zmk:    "Hello, world",
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Text","s":"Hello,"},{"t":"Space"},{"t":"Text","s":"world"}]`,
			encoderHTML:   "Hello, world",
			encoderNative: `Text "Hello,",Space,Text "world"`,
			encoderText:   "Hello, world",
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Italic formatting",
		zmk:    "//italic//",
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Italic","i":[{"t":"Text","s":"italic"}]}]`,
			encoderHTML:   "<i>italic</i>",
			encoderNative: `Italic [Text "italic"]`,
			encoderText:   "italic",
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Emphasized formatting",
		zmk:    "//emph//{-}",
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Emph","i":[{"t":"Text","s":"emph"}]}]`,
			encoderHTML:   "<em>emph</em>",
			encoderNative: `Emph [Text "emph"]`,
			encoderText:   "emph",
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Bold formatting",
		zmk:    "**bold**",
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Bold","i":[{"t":"Text","s":"bold"}]}]`,
			encoderHTML:   "<b>bold</b>",
			encoderNative: `Bold [Text "bold"]`,
			encoderText:   "bold",
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Strong formatting",
		zmk:    "**strong**{-}",
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Strong","i":[{"t":"Text","s":"strong"}]}]`,
			encoderHTML:   "<strong>strong</strong>",
			encoderNative: `Strong [Text "strong"]`,
			encoderText:   "strong",
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Underline formatting",
		zmk:    "__underline__",
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Underline","i":[{"t":"Text","s":"underline"}]}]`,
			encoderHTML:   "<u>underline</u>",
			encoderNative: `Underline [Text "underline"]`,
			encoderText:   "underline",
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Insert formatting",
		zmk:    "__insert__{-}",
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Insert","i":[{"t":"Text","s":"insert"}]}]`,
			encoderHTML:   "<ins>insert</ins>",
			encoderNative: `Insert [Text "insert"]`,
			encoderText:   "insert",
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Strike formatting",
		zmk:    "~~strike~~",
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Strikethrough","i":[{"t":"Text","s":"strike"}]}]`,
			encoderHTML:   "<s>strike</s>",
			encoderNative: `Strikethrough [Text "strike"]`,
			encoderText:   "strike",
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Delete formatting",
		zmk:    "~~delete~~{-}",
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Delete","i":[{"t":"Text","s":"delete"}]}]`,
			encoderHTML:   "<del>delete</del>",
			encoderNative: `Delete [Text "delete"]`,
			encoderText:   "delete",
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Update formatting",
		zmk:    "~~old~~{-}__new__{-}",
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Delete","i":[{"t":"Text","s":"old"}]},{"t":"Insert","i":[{"t":"Text","s":"new"}]}]`,
			encoderHTML:   "<del>old</del><ins>new</ins>",
			encoderNative: `Delete [Text "old"],Insert [Text "new"]`,
			encoderText:   "oldnew",
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Monospace formatting",
		zmk:    "''monospace''",
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Mono","i":[{"t":"Text","s":"monospace"}]}]`,
			encoderHTML:   `<span style="font-family:monospace">monospace</span>`,
			encoderNative: `Mono [Text "monospace"]`,
			encoderText:   "monospace",
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Superscript formatting",
		zmk:    "^^superscript^^",
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Super","i":[{"t":"Text","s":"superscript"}]}]`,
			encoderHTML:   `<sup>superscript</sup>`,
			encoderNative: `Super [Text "superscript"]`,
			encoderText:   `superscript`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Subscript formatting",
		zmk:    ",,subscript,,",
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Sub","i":[{"t":"Text","s":"subscript"}]}]`,
			encoderHTML:   `<sub>subscript</sub>`,
			encoderNative: `Sub [Text "subscript"]`,
			encoderText:   `subscript`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Quotes formatting",
		zmk:    `""quotes""`,
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Quote","i":[{"t":"Text","s":"quotes"}]}]`,
			encoderHTML:   `"quotes"`,
			encoderNative: `Quote [Text "quotes"]`,
			encoderText:   `quotes`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Quotes formatting (german)",
		zmk:    `""quotes""{lang=de}`,
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Quote","a":{"lang":"de"},"i":[{"t":"Text","s":"quotes"}]}]`,
			encoderHTML:   `<span lang="de">&bdquo;quotes&ldquo;</span>`,
			encoderNative: `Quote ("",[lang="de"]) [Text "quotes"]`,
			encoderText:   `quotes`,
			encoderZmk:    `""quotes""{lang="de"}`,
		},
	},
	{
		descr:  "Quotation formatting",
		zmk:    `<<quotation<<`,
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Quotation","i":[{"t":"Text","s":"quotation"}]}]`,
			encoderHTML:   `<q>quotation</q>`,
			encoderNative: `Quotation [Text "quotation"]`,
			encoderText:   `quotation`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Small formatting",
		zmk:    `;;small;;`,
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Small","i":[{"t":"Text","s":"small"}]}]`,
			encoderHTML:   `<small>small</small>`,
			encoderNative: `Small [Text "small"]`,
			encoderText:   `small`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Span formatting",
		zmk:    `::span::`,
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Span","i":[{"t":"Text","s":"span"}]}]`,
			encoderHTML:   `<span>span</span>`,
			encoderNative: `Span [Text "span"]`,
			encoderText:   `span`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Code formatting",
		zmk:    "``code``",
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Code","s":"code"}]`,
			encoderHTML:   `<code>code</code>`,
			encoderNative: `Code "code"`,
			encoderText:   `code`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Input formatting",
		zmk:    `++input++`,
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Input","s":"input"}]`,
			encoderHTML:   `<kbd>input</kbd>`,
			encoderNative: `Input "input"`,
			encoderText:   `input`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Output formatting",
		zmk:    `==output==`,
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Output","s":"output"}]`,
			encoderHTML:   `<samp>output</samp>`,
			encoderNative: `Output "output"`,
			encoderText:   `output`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Nested Span Quote formatting",
		zmk:    `::""abc""::{lang=fr}`,
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Span","a":{"lang":"fr"},"i":[{"t":"Quote","i":[{"t":"Text","s":"abc"}]}]}]`,
			encoderHTML:   `<span lang="fr">&laquo;&nbsp;abc&nbsp;&raquo;</span>`,
			encoderNative: `Span ("",[lang="fr"]) [Quote [Text "abc"]]`,
			encoderText:   `abc`,
			encoderZmk:    `::""abc""::{lang="fr"}`,
		},
	},
	{
		descr:  "Simple Citation",
		zmk:    `[@Stern18]`,
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Cite","s":"Stern18"}]`,
			encoderHTML:   `Stern18`, // TODO
			encoderNative: `Cite "Stern18"`,
			encoderText:   ``,
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "No comment",
		zmk:    `% comment`,
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Text","s":"%"},{"t":"Space"},{"t":"Text","s":"comment"}]`,
			encoderHTML:   `% comment`,
			encoderNative: `Text "%",Space,Text "comment"`,
			encoderText:   `% comment`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Line comment",
		zmk:    `%% line comment`,
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Comment","s":"line comment"}]`,
			encoderHTML:   `<!-- line comment -->`,
			encoderNative: `Comment "line comment"`,
			encoderText:   ``,
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Comment after text",
		zmk:    `Text %% comment`,
		inline: true,
		expect: expectMap{
			// TODO: Space before comment should be removed
			encoderDJSON: `[{"t":"Text","s":"Text"},{"t":"Space"},{"t":"Comment","s":"comment"}]`,
			encoderHTML:  `Text <!-- comment -->`,
			// TODO: Space before comment should be removed
			encoderNative: `Text "Text",Space,Comment "comment"`,
			// TODO: Space at end should be removed
			encoderText: `Text `,
			encoderZmk:  useZmk,
		},
	},
	{
		descr:  "Simple footnote",
		zmk:    `[^footnote]`,
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Footnote","i":[{"t":"Text","s":"footnote"}]}]`,
			encoderHTML:   `<sup id="fnref:0"><a href="#fn:0" class="zs-footnote-ref" role="doc-noteref">0</a></sup>`,
			encoderNative: `Footnote [Text "footnote"]`,
			encoderText:   ` footnote`, // TODO: remove leading space
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "Simple mark",
		zmk:    `[!mark]`,
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Mark","s":"mark"}]`,
			encoderHTML:   ``,
			encoderNative: `Mark "mark"`,
			encoderText:   ``,
			encoderZmk:    useZmk,
		},
	},
	{
		descr:  "",
		zmk:    ``,
		inline: true,
		expect: expectMap{
			encoderDJSON:  `[]`,
			encoderHTML:   ``,
			encoderNative: ``,
			encoderText:   ``,
			encoderZmk:    useZmk,
		},
	},
}

// func TestEncoderInline(t *testing.T) {
// 	executeTestCases(t, tcsInline)
// }
