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

var tcsInline = []zmkTestCase{
	{
		descr: "Empty Zettelmarkup should produce near nothing (inline)",
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
		descr: "Simple text: Hello, world (inline)",
		zmk:   `Hello, world`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Text","s":"Hello,"},{"t":"Space"},{"t":"Text","s":"world"}]`,
			encoderHTML:   "Hello, world",
			encoderNative: `Text "Hello,",Space,Text "world"`,
			encoderText:   "Hello, world",
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Emphasized formatting",
		zmk:   "__emph__",
		expect: expectMap{
			encoderDJSON:  `[{"t":"Emph","i":[{"t":"Text","s":"emph"}]}]`,
			encoderHTML:   "<em>emph</em>",
			encoderNative: `Emph [Text "emph"]`,
			encoderText:   "emph",
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Emphasized formatting (deprecated)",
		zmk:   "//emph//",
		expect: expectMap{
			encoderDJSON:  `[{"t":"Emph","i":[{"t":"Text","s":"emph"}]}]`,
			encoderHTML:   "<em>emph</em>",
			encoderNative: `Emph [Text "emph"]`,
			encoderText:   "emph",
			encoderZmk:    "__emph__",
		},
	},
	{
		descr: "Strong formatting",
		zmk:   "**strong**",
		expect: expectMap{
			encoderDJSON:  `[{"t":"Strong","i":[{"t":"Text","s":"strong"}]}]`,
			encoderHTML:   "<strong>strong</strong>",
			encoderNative: `Strong [Text "strong"]`,
			encoderText:   "strong",
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Insert formatting",
		zmk:   ">>insert>>",
		expect: expectMap{
			encoderDJSON:  `[{"t":"Insert","i":[{"t":"Text","s":"insert"}]}]`,
			encoderHTML:   "<ins>insert</ins>",
			encoderNative: `Insert [Text "insert"]`,
			encoderText:   "insert",
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Delete formatting",
		zmk:   "~~delete~~",
		expect: expectMap{
			encoderDJSON:  `[{"t":"Delete","i":[{"t":"Text","s":"delete"}]}]`,
			encoderHTML:   "<del>delete</del>",
			encoderNative: `Delete [Text "delete"]`,
			encoderText:   "delete",
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Update formatting",
		zmk:   "~~old~~>>new>>",
		expect: expectMap{
			encoderDJSON:  `[{"t":"Delete","i":[{"t":"Text","s":"old"}]},{"t":"Insert","i":[{"t":"Text","s":"new"}]}]`,
			encoderHTML:   "<del>old</del><ins>new</ins>",
			encoderNative: `Delete [Text "old"],Insert [Text "new"]`,
			encoderText:   "oldnew",
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Monospace formatting",
		zmk:   "''monospace''",
		expect: expectMap{
			encoderDJSON:  `[{"t":"Mono","i":[{"t":"Text","s":"monospace"}]}]`,
			encoderHTML:   `<span class="zs-monospace">monospace</span>`,
			encoderNative: `Mono [Text "monospace"]`,
			encoderText:   "monospace",
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Superscript formatting",
		zmk:   "^^superscript^^",
		expect: expectMap{
			encoderDJSON:  `[{"t":"Super","i":[{"t":"Text","s":"superscript"}]}]`,
			encoderHTML:   `<sup>superscript</sup>`,
			encoderNative: `Super [Text "superscript"]`,
			encoderText:   `superscript`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Subscript formatting",
		zmk:   ",,subscript,,",
		expect: expectMap{
			encoderDJSON:  `[{"t":"Sub","i":[{"t":"Text","s":"subscript"}]}]`,
			encoderHTML:   `<sub>subscript</sub>`,
			encoderNative: `Sub [Text "subscript"]`,
			encoderText:   `subscript`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Quotes formatting",
		zmk:   `""quotes""`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Quote","i":[{"t":"Text","s":"quotes"}]}]`,
			encoderHTML:   `"quotes"`,
			encoderNative: `Quote [Text "quotes"]`,
			encoderText:   `quotes`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Quotes formatting (german)",
		zmk:   `""quotes""{lang=de}`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Quote","a":{"lang":"de"},"i":[{"t":"Text","s":"quotes"}]}]`,
			encoderHTML:   `<span lang="de">&bdquo;quotes&ldquo;</span>`,
			encoderNative: `Quote ("",[lang="de"]) [Text "quotes"]`,
			encoderText:   `quotes`,
			encoderZmk:    `""quotes""{lang="de"}`,
		},
	},
	{
		descr: "Quotation formatting",
		zmk:   `<<quotation<<`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Quotation","i":[{"t":"Text","s":"quotation"}]}]`,
			encoderHTML:   `<q>quotation</q>`,
			encoderNative: `Quotation [Text "quotation"]`,
			encoderText:   `quotation`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Small formatting",
		zmk:   `;;small;;`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Small","i":[{"t":"Text","s":"small"}]}]`,
			encoderHTML:   `<small>small</small>`,
			encoderNative: `Small [Text "small"]`,
			encoderText:   `small`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Span formatting",
		zmk:   `::span::`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Span","i":[{"t":"Text","s":"span"}]}]`,
			encoderHTML:   `<span>span</span>`,
			encoderNative: `Span [Text "span"]`,
			encoderText:   `span`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Code formatting",
		zmk:   "``code``",
		expect: expectMap{
			encoderDJSON:  `[{"t":"Code","s":"code"}]`,
			encoderHTML:   `<code>code</code>`,
			encoderNative: `Code "code"`,
			encoderText:   `code`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Input formatting",
		zmk:   `++input++`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Input","s":"input"}]`,
			encoderHTML:   `<kbd>input</kbd>`,
			encoderNative: `Input "input"`,
			encoderText:   `input`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Output formatting",
		zmk:   `==output==`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Output","s":"output"}]`,
			encoderHTML:   `<samp>output</samp>`,
			encoderNative: `Output "output"`,
			encoderText:   `output`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Nested Span Quote formatting",
		zmk:   `::""abc""::{lang=fr}`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Span","a":{"lang":"fr"},"i":[{"t":"Quote","i":[{"t":"Text","s":"abc"}]}]}]`,
			encoderHTML:   `<span lang="fr">&laquo;&nbsp;abc&nbsp;&raquo;</span>`,
			encoderNative: `Span ("",[lang="fr"]) [Quote [Text "abc"]]`,
			encoderText:   `abc`,
			encoderZmk:    `::""abc""::{lang="fr"}`,
		},
	},
	{
		descr: "Simple Citation",
		zmk:   `[@Stern18]`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Cite","s":"Stern18"}]`,
			encoderHTML:   `Stern18`, // TODO
			encoderNative: `Cite "Stern18"`,
			encoderText:   ``,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "No comment",
		zmk:   `% comment`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Text","s":"%"},{"t":"Space"},{"t":"Text","s":"comment"}]`,
			encoderHTML:   `% comment`,
			encoderNative: `Text "%",Space,Text "comment"`,
			encoderText:   `% comment`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Line comment",
		zmk:   `%% line comment`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Comment","s":"line comment"}]`,
			encoderHTML:   `<!-- line comment -->`,
			encoderNative: `Comment "line comment"`,
			encoderText:   ``,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Comment after text",
		zmk:   `Text %% comment`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Text","s":"Text"},{"t":"Comment","s":"comment"}]`,
			encoderHTML:   `Text <!-- comment -->`,
			encoderNative: `Text "Text",Comment "comment"`,
			encoderText:   `Text`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Simple footnote",
		zmk:   `[^footnote]`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Footnote","i":[{"t":"Text","s":"footnote"}]}]`,
			encoderHTML:   `<sup id="fnref:0"><a href="#fn:0" class="zs-footnote-ref" role="doc-noteref">0</a></sup>`,
			encoderNative: `Footnote [Text "footnote"]`,
			encoderText:   `footnote`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Simple mark",
		zmk:   `[!mark]`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Mark","s":"mark"}]`,
			encoderHTML:   ``,
			encoderNative: `Mark "mark"`,
			encoderText:   ``,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Dummy Link",
		zmk:   `[[abc]]`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Link","q":"external","s":"abc","i":[{"t":"Text","s":"abc"}]}]`,
			encoderHTML:   `<a href="abc" class="zs-external">abc</a>`,
			encoderNative: `Link EXTERNAL "abc" []`,
			encoderText:   ``,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Simple URL",
		zmk:   `[[https://zettelstore.de]]`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Link","q":"external","s":"https://zettelstore.de","i":[{"t":"Text","s":"https://zettelstore.de"}]}]`,
			encoderHTML:   `<a href="https://zettelstore.de" class="zs-external">https://zettelstore.de</a>`,
			encoderNative: `Link EXTERNAL "https://zettelstore.de" []`,
			encoderText:   ``,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "URL with Text",
		zmk:   `[[Home|https://zettelstore.de]]`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Link","q":"external","s":"https://zettelstore.de","i":[{"t":"Text","s":"Home"}]}]`,
			encoderHTML:   `<a href="https://zettelstore.de" class="zs-external">Home</a>`,
			encoderNative: `Link EXTERNAL "https://zettelstore.de" [Text "Home"]`,
			encoderText:   `Home`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Simple Zettel ID",
		zmk:   `[[00000000000100]]`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Link","q":"zettel","s":"00000000000100","i":[{"t":"Text","s":"00000000000100"}]}]`,
			encoderHTML:   `<a href="00000000000100">00000000000100</a>`,
			encoderNative: `Link ZETTEL "00000000000100" []`,
			encoderText:   ``,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Zettel ID with Text",
		zmk:   `[[Config|00000000000100]]`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Link","q":"zettel","s":"00000000000100","i":[{"t":"Text","s":"Config"}]}]`,
			encoderHTML:   `<a href="00000000000100">Config</a>`,
			encoderNative: `Link ZETTEL "00000000000100" [Text "Config"]`,
			encoderText:   `Config`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Simple Zettel ID with fragment",
		zmk:   `[[00000000000100#frag]]`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Link","q":"zettel","s":"00000000000100#frag","i":[{"t":"Text","s":"00000000000100#frag"}]}]`,
			encoderHTML:   `<a href="00000000000100#frag">00000000000100#frag</a>`,
			encoderNative: `Link ZETTEL "00000000000100#frag" []`,
			encoderText:   ``,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Zettel ID with Text and fragment",
		zmk:   `[[Config|00000000000100#frag]]`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Link","q":"zettel","s":"00000000000100#frag","i":[{"t":"Text","s":"Config"}]}]`,
			encoderHTML:   `<a href="00000000000100#frag">Config</a>`,
			encoderNative: `Link ZETTEL "00000000000100#frag" [Text "Config"]`,
			encoderText:   `Config`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Fragment link to self",
		zmk:   `[[#frag]]`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Link","q":"self","s":"#frag","i":[{"t":"Text","s":"#frag"}]}]`,
			encoderHTML:   `<a href="#frag">#frag</a>`,
			encoderNative: `Link SELF "#frag" []`,
			encoderText:   ``,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Hosted link",
		zmk:   `[[H|/hosted]]`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Link","q":"local","s":"/hosted","i":[{"t":"Text","s":"H"}]}]`,
			encoderHTML:   `<a href="/hosted">H</a>`,
			encoderNative: `Link LOCAL "/hosted" [Text "H"]`,
			encoderText:   `H`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Based link",
		zmk:   `[[B|/based]]`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Link","q":"local","s":"/based","i":[{"t":"Text","s":"B"}]}]`,
			encoderHTML:   `<a href="/based">B</a>`,
			encoderNative: `Link LOCAL "/based" [Text "B"]`,
			encoderText:   `B`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Relative link",
		zmk:   `[[R|../relative]]`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Link","q":"local","s":"../relative","i":[{"t":"Text","s":"R"}]}]`,
			encoderHTML:   `<a href="../relative">R</a>`,
			encoderNative: `Link LOCAL "../relative" [Text "R"]`,
			encoderText:   `R`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Dummy Embed",
		zmk:   `{{abc}}`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Embed","s":"abc"}]`,
			encoderHTML:   `<img src="abc" alt="">`,
			encoderNative: `Embed EXTERNAL "abc"`,
			encoderText:   ``,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "",
		zmk:   ``,
		expect: expectMap{
			encoderDJSON:  `[]`,
			encoderHTML:   ``,
			encoderNative: ``,
			encoderText:   ``,
			encoderZmk:    useZmk,
		},
	},
}
