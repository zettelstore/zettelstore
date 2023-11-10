//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
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
			encoderSz:    `(INLINE)`,
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
			encoderSz:    `(INLINE (TEXT "Hello,") (SPACE) (TEXT "world"))`,
			encoderSHTML: `("Hello," " " "world")`,
			encoderText:  "Hello, world",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Soft Break",
		zmk:   "soft\nbreak",
		expect: expectMap{
			encoderHTML:  "soft break",
			encoderMD:    "soft\nbreak",
			encoderSz:    `(INLINE (TEXT "soft") (SOFT) (TEXT "break"))`,
			encoderSHTML: `("soft" " " "break")`,
			encoderText:  "soft break",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Hard Break",
		zmk:   "hard\\\nbreak",
		expect: expectMap{
			encoderHTML:  "hard<br>break",
			encoderMD:    "hard\\\nbreak",
			encoderSz:    `(INLINE (TEXT "hard") (HARD) (TEXT "break"))`,
			encoderSHTML: `("hard" (br) "break")`,
			encoderText:  "hard\nbreak",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Emphasized formatting",
		zmk:   "__emph__",
		expect: expectMap{
			encoderHTML:  "<em>emph</em>",
			encoderMD:    "*emph*",
			encoderSz:    `(INLINE (FORMAT-EMPH () (TEXT "emph")))`,
			encoderSHTML: `((em "emph"))`,
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
			encoderSz:    `(INLINE (FORMAT-STRONG () (TEXT "strong")))`,
			encoderSHTML: `((strong "strong"))`,
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
			encoderSz:    `(INLINE (FORMAT-INSERT () (TEXT "insert")))`,
			encoderSHTML: `((ins "insert"))`,
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
			encoderSz:    `(INLINE (FORMAT-DELETE () (TEXT "delete")))`,
			encoderSHTML: `((del "delete"))`,
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
			encoderSz:    `(INLINE (FORMAT-DELETE () (TEXT "old")) (FORMAT-INSERT () (TEXT "new")))`,
			encoderSHTML: `((del "old") (ins "new"))`,
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
			encoderSz:    `(INLINE (FORMAT-SUPER () (TEXT "superscript")))`,
			encoderSHTML: `((sup "superscript"))`,
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
			encoderSz:    `(INLINE (FORMAT-SUB () (TEXT "subscript")))`,
			encoderSHTML: `((sub "subscript"))`,
			encoderText:  `subscript`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Quotes formatting",
		zmk:   `""quotes""`,
		expect: expectMap{
			encoderHTML:  "<span>&ldquo;quotes&rdquo;</span>",
			encoderMD:    "<q>quotes</q>",
			encoderSz:    `(INLINE (FORMAT-QUOTE () (TEXT "quotes")))`,
			encoderSHTML: `((span (@H "&ldquo;") "quotes" (@H "&rdquo;")))`,
			encoderText:  `quotes`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Quotes formatting (german)",
		zmk:   `""quotes""{lang=de}`,
		expect: expectMap{
			encoderHTML:  `<span lang="de">&bdquo;quotes&ldquo;</span>`,
			encoderMD:    "<q>quotes</q>",
			encoderSz:    `(INLINE (FORMAT-QUOTE (quote (("lang" . "de"))) (TEXT "quotes")))`,
			encoderSHTML: `((span (@ (lang . "de")) (@H "&bdquo;") "quotes" (@H "&ldquo;")))`,
			encoderText:  `quotes`,
			encoderZmk:   `""quotes""{lang="de"}`,
		},
	},
	{
		descr: "Empty quotes (unknown)",
		zmk:   `""""{lang=unknown}`,
		expect: expectMap{
			encoderHTML:  `<span lang="unknown">&quot;&quot;</span>`,
			encoderMD:    "<q></q>",
			encoderSz:    `(INLINE (FORMAT-QUOTE (quote (("lang" . "unknown")))))`,
			encoderSHTML: `((span (@ (lang . "unknown")) (@H "&quot;" "&quot;")))`,
			encoderText:  ``,
			encoderZmk:   `""""{lang="unknown"}`,
		},
	},
	{
		descr: "Span formatting",
		zmk:   `::span::`,
		expect: expectMap{
			encoderHTML:  `<span>span</span>`,
			encoderMD:    "span",
			encoderSz:    `(INLINE (FORMAT-SPAN () (TEXT "span")))`,
			encoderSHTML: `((span "span"))`,
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
			encoderSz:    `(INLINE (LITERAL-CODE () "code"))`,
			encoderSHTML: `((code "code"))`,
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
			encoderSz:    `(INLINE (LITERAL-CODE (quote (("-" . ""))) "x y"))`,
			encoderSHTML: "((code \"x\u2423y\"))",
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
			encoderSz:    `(INLINE (LITERAL-CODE () "<script ") (SPACE) (TEXT "abc"))`,
			encoderSHTML: `((code "<script ") " " "abc")`,
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
			encoderSz:    `(INLINE (LITERAL-INPUT () "input"))`,
			encoderSHTML: `((kbd "input"))`,
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
			encoderSz:    `(INLINE (LITERAL-OUTPUT () "output"))`,
			encoderSHTML: `((samp "output"))`,
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
			encoderSz:    `(INLINE (LITERAL-MATH () "\\TeX"))`,
			encoderSHTML: `((code (@ (class . "zs-math")) "\\TeX"))`,
			encoderText:  `\TeX`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Nested Span Quote formatting",
		zmk:   `::""abc""::{lang=fr}`,
		expect: expectMap{
			encoderHTML:  `<span lang="fr"><span>&laquo;&nbsp;abc&nbsp;&raquo;</span></span>`,
			encoderMD:    "<q>abc</q>",
			encoderSz:    `(INLINE (FORMAT-SPAN (quote (("lang" . "fr"))) (FORMAT-QUOTE () (TEXT "abc"))))`,
			encoderSHTML: `((span (@ (lang . "fr")) (span (@H "&laquo;&nbsp;") "abc" (@H "&nbsp;&raquo;"))))`,
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
			encoderSz:    `(INLINE (CITE () "Stern18"))`,
			encoderSHTML: `((span "Stern18"))`, // TODO
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Citation",
		zmk:   `[@Stern18 p.23]`,
		expect: expectMap{
			encoderHTML:  `<span>Stern18, p.23</span>`, // TODO
			encoderMD:    "p.23",
			encoderSz:    `(INLINE (CITE () "Stern18" (TEXT "p.23")))`,
			encoderSHTML: `((span "Stern18" ", " "p.23"))`, // TODO
			encoderText:  `p.23`,
			encoderZmk:   useZmk,
		},
	}, {
		descr: "No comment",
		zmk:   `% comment`,
		expect: expectMap{
			encoderHTML:  `% comment`,
			encoderMD:    "% comment",
			encoderSz:    `(INLINE (TEXT "%") (SPACE) (TEXT "comment"))`,
			encoderSHTML: `("%" " " "comment")`,
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
			encoderSz:    `(INLINE (LITERAL-COMMENT () "line comment"))`,
			encoderSHTML: `(())`,
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
			encoderSz:    `(INLINE (LITERAL-COMMENT (quote (("-" . ""))) "line comment"))`,
			encoderSHTML: `((@@ "line comment"))`,
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
			encoderSz:    `(INLINE (TEXT "Text") (LITERAL-COMMENT (quote (("-" . ""))) "comment"))`,
			encoderSHTML: `("Text" (@@ "comment"))`,
			encoderText:  `Text`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Comment after text and with -->",
		zmk:   `Text %%{-} comment --> end`,
		expect: expectMap{
			encoderHTML:  `Text<!-- comment -&#45;> end -->`,
			encoderMD:    "Text",
			encoderSz:    `(INLINE (TEXT "Text") (LITERAL-COMMENT (quote (("-" . ""))) "comment --> end"))`,
			encoderSHTML: `("Text" (@@ "comment --> end"))`,
			encoderText:  `Text`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple inline endnote",
		zmk:   `[^endnote]`,
		expect: expectMap{
			encoderHTML:  `<sup id="fnref:1"><a class="zs-noteref" href="#fn:1" role="doc-noteref">1</a></sup>`,
			encoderMD:    "",
			encoderSz:    `(INLINE (ENDNOTE () (quote (INLINE (TEXT "endnote")))))`,
			encoderSHTML: `((sup (@ (id . "fnref:1")) (a (@ (class . "zs-noteref") (href . "#fn:1") (role . "doc-noteref")) "1")))`,
			encoderText:  `endnote`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple mark",
		zmk:   `[!mark]`,
		expect: expectMap{
			encoderHTML:  `<a id="mark"></a>`,
			encoderMD:    "",
			encoderSz:    `(INLINE (MARK "mark" "mark" "mark"))`,
			encoderSHTML: `((a (@ (id . "mark"))))`,
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
			encoderSz:    `(INLINE (MARK "mark" "mark" "mark" (TEXT "with") (SPACE) (TEXT "text")))`,
			encoderSHTML: `((a (@ (id . "mark")) "with" " " "text"))`,
			encoderText:  `with text`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Invalid Link",
		zmk:   `[[link|00000000000000]]`,
		expect: expectMap{
			encoderHTML:  `<span>link</span>`,
			encoderMD:    "[link](00000000000000)",
			encoderSz:    `(INLINE (LINK-INVALID () "00000000000000" (TEXT "link")))`,
			encoderSHTML: `((span "link"))`,
			encoderText:  `link`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Invalid Simple Link",
		zmk:   `[[00000000000000]]`,
		expect: expectMap{
			encoderHTML:  `<span>00000000000000</span>`,
			encoderMD:    "[00000000000000](00000000000000)",
			encoderSz:    `(INLINE (LINK-INVALID () "00000000000000"))`,
			encoderSHTML: `((span "00000000000000"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Dummy Link",
		zmk:   `[[abc]]`,
		expect: expectMap{
			encoderHTML:  `<a class="external" href="abc">abc</a>`,
			encoderMD:    "[abc](abc)",
			encoderSz:    `(INLINE (LINK-EXTERNAL () "abc"))`,
			encoderSHTML: `((a (@ (class . "external") (href . "abc")) "abc"))`,
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
			encoderSz:    `(INLINE (LINK-EXTERNAL () "https://zettelstore.de"))`,
			encoderSHTML: `((a (@ (class . "external") (href . "https://zettelstore.de")) "https://zettelstore.de"))`,
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
			encoderSz:    `(INLINE (LINK-EXTERNAL () "https://zettelstore.de" (TEXT "Home")))`,
			encoderSHTML: `((a (@ (class . "external") (href . "https://zettelstore.de")) "Home"))`,
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
			encoderSz:    `(INLINE (LINK-ZETTEL () "00000000000100"))`,
			encoderSHTML: `((a (@ (href . "00000000000100")) "00000000000100"))`,
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
			encoderSz:    `(INLINE (LINK-ZETTEL () "00000000000100" (TEXT "Config")))`,
			encoderSHTML: `((a (@ (href . "00000000000100")) "Config"))`,
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
			encoderSz:    `(INLINE (LINK-ZETTEL () "00000000000100#frag"))`,
			encoderSHTML: `((a (@ (href . "00000000000100#frag")) "00000000000100#frag"))`,
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
			encoderSz:    `(INLINE (LINK-ZETTEL () "00000000000100#frag" (TEXT "Config")))`,
			encoderSHTML: `((a (@ (href . "00000000000100#frag")) "Config"))`,
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
			encoderSz:    `(INLINE (LINK-SELF () "#frag"))`,
			encoderSHTML: `((a (@ (href . "#frag")) "#frag"))`,
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
			encoderSz:    `(INLINE (LINK-HOSTED () "/hosted" (TEXT "H")))`,
			encoderSHTML: `((a (@ (href . "/hosted")) "H"))`,
			encoderText:  `H`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Based link",
		zmk:   `[[B|//based]]`,
		expect: expectMap{
			encoderHTML:  `<a href="/based">B</a>`,
			encoderMD:    "[B](/based)",
			encoderSz:    `(INLINE (LINK-BASED () "/based" (TEXT "B")))`,
			encoderText:  `B`,
			encoderSHTML: `((a (@ (href . "/based")) "B"))`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Relative link",
		zmk:   `[[R|../relative]]`,
		expect: expectMap{
			encoderHTML:  `<a href="../relative">R</a>`,
			encoderMD:    "[R](../relative)",
			encoderSz:    `(INLINE (LINK-HOSTED () "../relative" (TEXT "R")))`,
			encoderSHTML: `((a (@ (href . "../relative")) "R"))`,
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
			encoderSz:    `(INLINE (LINK-QUERY () "title:syntax"))`,
			encoderSHTML: `((a (@ (href . "?q=title%3Asyntax")) "title:syntax"))`,
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
			encoderSz:    `(INLINE (LINK-QUERY () "title:syntax" (TEXT "Q")))`,
			encoderSHTML: `((a (@ (href . "?q=title%3Asyntax")) "Q"))`,
			encoderText:  `Q`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Dummy Embed",
		zmk:   `{{abc}}`,
		expect: expectMap{
			encoderHTML:  `<img src="abc">`,
			encoderMD:    "![abc](abc)",
			encoderSz:    `(INLINE (EMBED () (quote (EXTERNAL "abc")) ""))`,
			encoderSHTML: `((img (@ (src . "abc"))))`,
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
			encoderSz:    `(INLINE)`,
			encoderSHTML: `()`,
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
			encoderSz:    `(INLINE (LITERAL-ZETTEL (quote (("" . "text"))) "<hr>"))`,
			encoderSHTML: `(())`,
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
			encoderSz:    `(INLINE)`,
			encoderSHTML: `()`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
}
