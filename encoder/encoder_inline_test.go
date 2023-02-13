//-----------------------------------------------------------------------------
// Copyright (c) 2021-2023 Detlef Stern
//
// This file is part of Zettelstore.
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
			encoderHTML:  "",
			encoderMD:    "",
			encoderSexpr: `()`,
			encoderSHTML: `()`,
			encoderText:  "",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple text: Hello, world (inline)",
		zmk:   `Hello, world`,
		expect: expectMap{
			encoderHTML:  "Hello, world",
			encoderMD:    "Hello, world",
			encoderSexpr: `((TEXT "Hello,") (SPACE) (TEXT "world"))`,
			encoderText:  "Hello, world",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Emphasized formatting",
		zmk:   "__emph__",
		expect: expectMap{
			encoderHTML:  "<em>emph</em>",
			encoderMD:    "*emph*",
			encoderSexpr: `((FORMAT-EMPH () (TEXT "emph")))`,
			encoderText:  "emph",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Strong formatting",
		zmk:   "**strong**",
		expect: expectMap{
			encoderHTML:  "<strong>strong</strong>",
			encoderMD:    "__strong__",
			encoderSexpr: `((FORMAT-STRONG () (TEXT "strong")))`,
			encoderText:  "strong",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Insert formatting",
		zmk:   ">>insert>>",
		expect: expectMap{
			encoderHTML:  "<ins>insert</ins>",
			encoderMD:    "insert",
			encoderSexpr: `((FORMAT-INSERT () (TEXT "insert")))`,
			encoderText:  "insert",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Delete formatting",
		zmk:   "~~delete~~",
		expect: expectMap{
			encoderHTML:  "<del>delete</del>",
			encoderMD:    "delete",
			encoderSexpr: `((FORMAT-DELETE () (TEXT "delete")))`,
			encoderText:  "delete",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Update formatting",
		zmk:   "~~old~~>>new>>",
		expect: expectMap{
			encoderHTML:  "<del>old</del><ins>new</ins>",
			encoderMD:    "oldnew",
			encoderSexpr: `((FORMAT-DELETE () (TEXT "old")) (FORMAT-INSERT () (TEXT "new")))`,
			encoderText:  "oldnew",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Superscript formatting",
		zmk:   "^^superscript^^",
		expect: expectMap{
			encoderHTML:  `<sup>superscript</sup>`,
			encoderMD:    "superscript",
			encoderSexpr: `((FORMAT-SUPER () (TEXT "superscript")))`,
			encoderText:  `superscript`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Subscript formatting",
		zmk:   ",,subscript,,",
		expect: expectMap{
			encoderHTML:  `<sub>subscript</sub>`,
			encoderMD:    "subscript",
			encoderSexpr: `((FORMAT-SUB () (TEXT "subscript")))`,
			encoderText:  `subscript`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Quotes formatting",
		zmk:   `""quotes""`,
		expect: expectMap{
			encoderHTML:  "<q>quotes</q>",
			encoderMD:    "<q>quotes</q>",
			encoderSexpr: `((FORMAT-QUOTE () (TEXT "quotes")))`,
			encoderText:  `quotes`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Quotes formatting (german)",
		zmk:   `""quotes""{lang=de}`,
		expect: expectMap{
			encoderHTML:  `<span lang="de"><q>quotes</q></span>`,
			encoderMD:    "<q>quotes</q>",
			encoderSexpr: `((FORMAT-QUOTE (("lang" "de")) (TEXT "quotes")))`,
			encoderText:  `quotes`,
			encoderZmk:   `""quotes""{lang="de"}`,
		},
	},
	{
		descr: "Span formatting",
		zmk:   `::span::`,
		expect: expectMap{
			encoderHTML:  `<span>span</span>`,
			encoderMD:    "span",
			encoderSexpr: `((FORMAT-SPAN () (TEXT "span")))`,
			encoderText:  `span`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Code formatting",
		zmk:   "``code``",
		expect: expectMap{
			encoderHTML:  `<code>code</code>`,
			encoderMD:    "`code`",
			encoderSexpr: `((LITERAL-CODE () "code"))`,
			encoderText:  `code`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Code formatting with visible space",
		zmk:   "``x y``{-}",
		expect: expectMap{
			encoderHTML:  "<code>x\u2423y</code>",
			encoderMD:    "`x y`",
			encoderSexpr: `((LITERAL-CODE (("-" "")) "x y"))`,
			encoderText:  `x y`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "HTML in Code formatting",
		zmk:   "``<script `` abc",
		expect: expectMap{
			encoderHTML:  "<code>&lt;script </code> abc",
			encoderMD:    "`<script ` abc",
			encoderSexpr: `((LITERAL-CODE () "<script ") (SPACE) (TEXT "abc"))`,
			encoderText:  `<script  abc`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Input formatting",
		zmk:   `''input''`,
		expect: expectMap{
			encoderHTML:  `<kbd>input</kbd>`,
			encoderMD:    "input",
			encoderSexpr: `((LITERAL-INPUT () "input"))`,
			encoderText:  `input`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Output formatting",
		zmk:   `==output==`,
		expect: expectMap{
			encoderHTML:  `<samp>output</samp>`,
			encoderMD:    "output",
			encoderSexpr: `((LITERAL-OUTPUT () "output"))`,
			encoderText:  `output`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Math formatting",
		zmk:   `$$\TeX$$`,
		expect: expectMap{
			encoderHTML:  `<code class="zs-math">\TeX</code>`,
			encoderMD:    "\\TeX",
			encoderSexpr: `((LITERAL-MATH () "\\TeX"))`,
			encoderText:  `\TeX`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Nested Span Quote formatting",
		zmk:   `::""abc""::{lang=fr}`,
		expect: expectMap{
			encoderHTML:  `<span lang="fr"><q>abc</q></span>`,
			encoderMD:    "<q>abc</q>",
			encoderSexpr: `((FORMAT-SPAN (("lang" "fr")) (FORMAT-QUOTE () (TEXT "abc"))))`,
			encoderText:  `abc`,
			encoderZmk:   `::""abc""::{lang="fr"}`,
		},
	},
	{
		descr: "Simple Citation",
		zmk:   `[@Stern18]`,
		expect: expectMap{
			encoderHTML:  `<span>Stern18</span>`, // TODO
			encoderMD:    "",
			encoderSexpr: `((CITE () "Stern18"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "No comment",
		zmk:   `% comment`,
		expect: expectMap{
			encoderHTML:  `% comment`,
			encoderMD:    "% comment",
			encoderSexpr: `((TEXT "%") (SPACE) (TEXT "comment"))`,
			encoderText:  `% comment`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Line comment (nogen HTML)",
		zmk:   `%% line comment`,
		expect: expectMap{
			encoderHTML:  ``,
			encoderMD:    "",
			encoderSexpr: `((LITERAL-COMMENT () "line comment"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Line comment",
		zmk:   `%%{-} line comment`,
		expect: expectMap{
			encoderHTML:  `<!-- line comment -->`,
			encoderMD:    "",
			encoderSexpr: `((LITERAL-COMMENT (("-" "")) "line comment"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Comment after text",
		zmk:   `Text %%{-} comment`,
		expect: expectMap{
			encoderHTML:  `Text<!-- comment -->`,
			encoderMD:    "Text",
			encoderSexpr: `((TEXT "Text") (LITERAL-COMMENT (("-" "")) "comment"))`,
			encoderText:  `Text`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Comment after text and with -->",
		zmk:   `Text %%{-} comment --> end`,
		expect: expectMap{
			encoderHTML:  `Text<!-- comment --&gt; end -->`,
			encoderMD:    "Text",
			encoderSexpr: `((TEXT "Text") (LITERAL-COMMENT (("-" "")) "comment --> end"))`,
			encoderText:  `Text`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple footnote",
		zmk:   `[^footnote]`,
		expect: expectMap{
			encoderHTML:  `<sup id="fnref:1"><a class="zs-noteref" href="#fn:1" role="doc-noteref">1</a></sup>`,
			encoderMD:    "",
			encoderSexpr: `((FOOTNOTE () (TEXT "footnote")))`,
			encoderText:  `footnote`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple mark",
		zmk:   `[!mark]`,
		expect: expectMap{
			encoderHTML:  `<a id="mark"></a>`,
			encoderMD:    "",
			encoderSexpr: `((MARK "mark" "mark" "mark"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Mark with text",
		zmk:   `[!mark|with text]`,
		expect: expectMap{
			encoderHTML:  `<a id="mark">with text</a>`,
			encoderMD:    "with text",
			encoderSexpr: `((MARK "mark" "mark" "mark" (TEXT "with") (SPACE) (TEXT "text")))`,
			encoderText:  `with text`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Dummy Link",
		zmk:   `[[abc]]`,
		expect: expectMap{
			encoderHTML:  `<a class="external" href="abc">abc</a>`,
			encoderMD:    "[abc](abc)",
			encoderSexpr: `((LINK-EXTERNAL () "abc"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple URL",
		zmk:   `[[https://zettelstore.de]]`,
		expect: expectMap{
			encoderHTML:  `<a class="external" href="https://zettelstore.de">https://zettelstore.de</a>`,
			encoderMD:    "<https://zettelstore.de>",
			encoderSexpr: `((LINK-EXTERNAL () "https://zettelstore.de"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "URL with Text",
		zmk:   `[[Home|https://zettelstore.de]]`,
		expect: expectMap{
			encoderHTML:  `<a class="external" href="https://zettelstore.de">Home</a>`,
			encoderMD:    "[Home](https://zettelstore.de)",
			encoderSexpr: `((LINK-EXTERNAL () "https://zettelstore.de" (TEXT "Home")))`,
			encoderText:  `Home`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple Zettel ID",
		zmk:   `[[00000000000100]]`,
		expect: expectMap{
			encoderHTML:  `<a href="00000000000100">00000000000100</a>`,
			encoderMD:    "[00000000000100](00000000000100)",
			encoderSexpr: `((LINK-ZETTEL () "00000000000100"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Zettel ID with Text",
		zmk:   `[[Config|00000000000100]]`,
		expect: expectMap{
			encoderHTML:  `<a href="00000000000100">Config</a>`,
			encoderMD:    "[Config](00000000000100)",
			encoderSexpr: `((LINK-ZETTEL () "00000000000100" (TEXT "Config")))`,
			encoderText:  `Config`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple Zettel ID with fragment",
		zmk:   `[[00000000000100#frag]]`,
		expect: expectMap{
			encoderHTML:  `<a href="00000000000100#frag">00000000000100#frag</a>`,
			encoderMD:    "[00000000000100#frag](00000000000100#frag)",
			encoderSexpr: `((LINK-ZETTEL () "00000000000100#frag"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Zettel ID with Text and fragment",
		zmk:   `[[Config|00000000000100#frag]]`,
		expect: expectMap{
			encoderHTML:  `<a href="00000000000100#frag">Config</a>`,
			encoderMD:    "[Config](00000000000100#frag)",
			encoderSexpr: `((LINK-ZETTEL () "00000000000100#frag" (TEXT "Config")))`,
			encoderText:  `Config`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Fragment link to self",
		zmk:   `[[#frag]]`,
		expect: expectMap{
			encoderHTML:  `<a href="#frag">#frag</a>`,
			encoderMD:    "[#frag](#frag)",
			encoderSexpr: `((LINK-SELF () "#frag"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Hosted link",
		zmk:   `[[H|/hosted]]`,
		expect: expectMap{
			encoderHTML:  `<a href="/hosted">H</a>`,
			encoderMD:    "[H](/hosted)",
			encoderSexpr: `((LINK-HOSTED () "/hosted" (TEXT "H")))`,
			encoderText:  `H`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Based link",
		zmk:   `[[B|/based]]`,
		expect: expectMap{
			encoderHTML:  `<a href="/based">B</a>`,
			encoderMD:    "[B](/based)",
			encoderSexpr: `((LINK-HOSTED () "/based" (TEXT "B")))`,
			encoderText:  `B`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Relative link",
		zmk:   `[[R|../relative]]`,
		expect: expectMap{
			encoderHTML:  `<a href="../relative">R</a>`,
			encoderMD:    "[R](../relative)",
			encoderSexpr: `((LINK-HOSTED () "../relative" (TEXT "R")))`,
			encoderText:  `R`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Query link w/o text",
		zmk:   `[[query:title:syntax]]`,
		expect: expectMap{
			encoderHTML:  `<a href="?q=title%3Asyntax">title:syntax</a>`,
			encoderMD:    "",
			encoderSexpr: `((LINK-QUERY () "title:syntax"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Query link with text",
		zmk:   `[[Q|query:title:syntax]]`,
		expect: expectMap{
			encoderHTML:  `<a href="?q=title%3Asyntax">Q</a>`,
			encoderMD:    "Q",
			encoderSexpr: `((LINK-QUERY () "title:syntax" (TEXT "Q")))`,
			encoderText:  `Q`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Dummy Embed",
		zmk:   `{{abc}}`,
		expect: expectMap{
			encoderHTML:  `<img alt="alternate description missing" src="abc">`,
			encoderMD:    "![abc](abc)",
			encoderSexpr: `((EMBED () (EXTERNAL "abc") ""))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Inline HTML Zettel",
		zmk:   `@@<hr>@@{="html"}`,
		expect: expectMap{
			encoderHTML:  ``,
			encoderMD:    "",
			encoderSexpr: `()`,
			encoderText:  ``,
			encoderZmk:   ``,
		},
	},
	{
		descr: "Inline Text Zettel",
		zmk:   `@@<hr>@@{="text"}`,
		expect: expectMap{
			encoderHTML:  ``,
			encoderMD:    "<hr>",
			encoderSexpr: `((LITERAL-ZETTEL (("" "text")) "<hr>"))`,
			encoderText:  `<hr>`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "",
		zmk:   ``,
		expect: expectMap{
			encoderHTML:  ``,
			encoderMD:    "",
			encoderSexpr: `()`,
			encoderSHTML: `()`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
}
