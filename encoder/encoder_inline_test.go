//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
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
			encoderZJSON: `[]`,
			encoderHTML:  "",
			encoderSexpr: `()`,
			encoderText:  "",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple text: Hello, world (inline)",
		zmk:   `Hello, world`,
		expect: expectMap{
			encoderZJSON: `[{"":"Text","s":"Hello,"},{"":"Space"},{"":"Text","s":"world"}]`,
			encoderHTML:  "Hello, world",
			encoderSexpr: `((TEXT "Hello,") (SPACE) (TEXT "world"))`,
			encoderText:  "Hello, world",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Emphasized formatting",
		zmk:   "__emph__",
		expect: expectMap{
			encoderZJSON: `[{"":"Emph","i":[{"":"Text","s":"emph"}]}]`,
			encoderHTML:  "<em>emph</em>",
			encoderSexpr: `((FORMAT-EMPH () (TEXT "emph")))`,
			encoderText:  "emph",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Strong formatting",
		zmk:   "**strong**",
		expect: expectMap{
			encoderZJSON: `[{"":"Strong","i":[{"":"Text","s":"strong"}]}]`,
			encoderHTML:  "<strong>strong</strong>",
			encoderSexpr: `((FORMAT-STRONG () (TEXT "strong")))`,
			encoderText:  "strong",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Insert formatting",
		zmk:   ">>insert>>",
		expect: expectMap{
			encoderZJSON: `[{"":"Insert","i":[{"":"Text","s":"insert"}]}]`,
			encoderHTML:  "<ins>insert</ins>",
			encoderSexpr: `((FORMAT-INSERT () (TEXT "insert")))`,
			encoderText:  "insert",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Delete formatting",
		zmk:   "~~delete~~",
		expect: expectMap{
			encoderZJSON: `[{"":"Delete","i":[{"":"Text","s":"delete"}]}]`,
			encoderHTML:  "<del>delete</del>",
			encoderSexpr: `((FORMAT-DELETE () (TEXT "delete")))`,
			encoderText:  "delete",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Update formatting",
		zmk:   "~~old~~>>new>>",
		expect: expectMap{
			encoderZJSON: `[{"":"Delete","i":[{"":"Text","s":"old"}]},{"":"Insert","i":[{"":"Text","s":"new"}]}]`,
			encoderHTML:  "<del>old</del><ins>new</ins>",
			encoderSexpr: `((FORMAT-DELETE () (TEXT "old")) (FORMAT-INSERT () (TEXT "new")))`,
			encoderText:  "oldnew",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Superscript formatting",
		zmk:   "^^superscript^^",
		expect: expectMap{
			encoderZJSON: `[{"":"Super","i":[{"":"Text","s":"superscript"}]}]`,
			encoderHTML:  `<sup>superscript</sup>`,
			encoderSexpr: `((FORMAT-SUPER () (TEXT "superscript")))`,
			encoderText:  `superscript`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Subscript formatting",
		zmk:   ",,subscript,,",
		expect: expectMap{
			encoderZJSON: `[{"":"Sub","i":[{"":"Text","s":"subscript"}]}]`,
			encoderHTML:  `<sub>subscript</sub>`,
			encoderSexpr: `((FORMAT-SUB () (TEXT "subscript")))`,
			encoderText:  `subscript`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Quotes formatting",
		zmk:   `""quotes""`,
		expect: expectMap{
			encoderZJSON: `[{"":"Quote","i":[{"":"Text","s":"quotes"}]}]`,
			encoderHTML:  "<q>quotes</q>",
			encoderSexpr: `((FORMAT-QUOTE () (TEXT "quotes")))`,
			encoderText:  `quotes`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Quotes formatting (german)",
		zmk:   `""quotes""{lang=de}`,
		expect: expectMap{
			encoderZJSON: `[{"":"Quote","a":{"lang":"de"},"i":[{"":"Text","s":"quotes"}]}]`,
			encoderHTML:  `<q lang="de">quotes</q>`,
			encoderSexpr: `((FORMAT-QUOTE (("lang" "de")) (TEXT "quotes")))`,
			encoderText:  `quotes`,
			encoderZmk:   `""quotes""{lang="de"}`,
		},
	},
	{
		descr: "Span formatting",
		zmk:   `::span::`,
		expect: expectMap{
			encoderZJSON: `[{"":"Span","i":[{"":"Text","s":"span"}]}]`,
			encoderHTML:  `<span>span</span>`,
			encoderSexpr: `((FORMAT-SPAN () (TEXT "span")))`,
			encoderText:  `span`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Code formatting",
		zmk:   "``code``",
		expect: expectMap{
			encoderZJSON: `[{"":"Code","s":"code"}]`,
			encoderHTML:  `<code>code</code>`,
			encoderSexpr: `((LITERAL-CODE () "code"))`,
			encoderText:  `code`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Code formatting with visible space",
		zmk:   "``x y``{-}",
		expect: expectMap{
			encoderZJSON: `[{"":"Code","a":{"-":""},"s":"x y"}]`,
			encoderHTML:  "<code>x\u2423y</code>",
			encoderSexpr: `((LITERAL-CODE (("-" "")) "x y"))`,
			encoderText:  `x y`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "HTML in Code formatting",
		zmk:   "``<script `` abc",
		expect: expectMap{
			encoderZJSON: `[{"":"Code","s":"<script "},{"":"Space"},{"":"Text","s":"abc"}]`,
			encoderHTML:  "<code>&lt;script\u00a0</code> abc",
			encoderSexpr: `((LITERAL-CODE () "<script ") (SPACE) (TEXT "abc"))`,
			encoderText:  `<script  abc`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Input formatting",
		zmk:   `''input''`,
		expect: expectMap{
			encoderZJSON: `[{"":"Input","s":"input"}]`,
			encoderHTML:  `<kbd>input</kbd>`,
			encoderSexpr: `((LITERAL-INPUT () "input"))`,
			encoderText:  `input`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Output formatting",
		zmk:   `==output==`,
		expect: expectMap{
			encoderZJSON: `[{"":"Output","s":"output"}]`,
			encoderHTML:  `<samp>output</samp>`,
			encoderSexpr: `((LITERAL-OUTPUT () "output"))`,
			encoderText:  `output`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Math formatting",
		zmk:   `$$\TeX$$`,
		expect: expectMap{
			encoderZJSON: `[{"":"Math","s":"\\TeX"}]`,
			encoderHTML:  `<code class="zs-math">\TeX</code>`,
			encoderSexpr: `((LITERAL-MATH () "\\TeX"))`,
			encoderText:  `\TeX`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Nested Span Quote formatting",
		zmk:   `::""abc""::{lang=fr}`,
		expect: expectMap{
			encoderZJSON: `[{"":"Span","a":{"lang":"fr"},"i":[{"":"Quote","i":[{"":"Text","s":"abc"}]}]}]`,
			encoderHTML:  `<span lang="fr"><q>abc</q></span>`,
			encoderSexpr: `((FORMAT-SPAN (("lang" "fr")) (FORMAT-QUOTE () (TEXT "abc"))))`,
			encoderText:  `abc`,
			encoderZmk:   `::""abc""::{lang="fr"}`,
		},
	},
	{
		descr: "Simple Citation",
		zmk:   `[@Stern18]`,
		expect: expectMap{
			encoderZJSON: `[{"":"Cite","s":"Stern18"}]`,
			encoderHTML:  `<span>Stern18</span>`, // TODO
			encoderSexpr: `((CITE () "Stern18"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "No comment",
		zmk:   `% comment`,
		expect: expectMap{
			encoderZJSON: `[{"":"Text","s":"%"},{"":"Space"},{"":"Text","s":"comment"}]`,
			encoderHTML:  `% comment`,
			encoderSexpr: `((TEXT "%") (SPACE) (TEXT "comment"))`,
			encoderText:  `% comment`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Line comment (nogen HTML)",
		zmk:   `%% line comment`,
		expect: expectMap{
			encoderZJSON: `[{"":"Comment","s":"line comment"}]`,
			encoderHTML:  ``,
			encoderSexpr: `((LITERAL-COMMENT () "line comment"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Line comment",
		zmk:   `%%{-} line comment`,
		expect: expectMap{
			encoderZJSON: `[{"":"Comment","a":{"-":""},"s":"line comment"}]`,
			encoderHTML:  `<!-- line comment -->`,
			encoderSexpr: `((LITERAL-COMMENT (("-" "")) "line comment"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Comment after text",
		zmk:   `Text %%{-} comment`,
		expect: expectMap{
			encoderZJSON: `[{"":"Text","s":"Text"},{"":"Comment","a":{"-":""},"s":"comment"}]`,
			encoderHTML:  `Text<!-- comment -->`,
			encoderSexpr: `((TEXT "Text") (LITERAL-COMMENT (("-" "")) "comment"))`,
			encoderText:  `Text`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Comment after text and with -->",
		zmk:   `Text %%{-} comment --> end`,
		expect: expectMap{
			encoderZJSON: `[{"":"Text","s":"Text"},{"":"Comment","a":{"-":""},"s":"comment --> end"}]`,
			encoderHTML:  `Text<!-- comment --&gt; end -->`,
			encoderSexpr: `((TEXT "Text") (LITERAL-COMMENT (("-" "")) "comment --> end"))`,
			encoderText:  `Text`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple footnote",
		zmk:   `[^footnote]`,
		expect: expectMap{
			encoderZJSON: `[{"":"Footnote","i":[{"":"Text","s":"footnote"}]}]`,
			encoderHTML:  `<sup id="fnref:1"><a class="zs-noteref" href="#fn:1" role="doc-noteref">1</a></sup>`,
			encoderSexpr: `((FOOTNOTE () (TEXT "footnote")))`,
			encoderText:  `footnote`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple mark",
		zmk:   `[!mark]`,
		expect: expectMap{
			encoderZJSON: `[{"":"Mark","s":"mark","q":"mark"}]`,
			encoderHTML:  `<a id="mark"></a>`,
			encoderSexpr: `((MARK "mark" "mark" "mark"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Mark with text",
		zmk:   `[!mark|with text]`,
		expect: expectMap{
			encoderZJSON: `[{"":"Mark","s":"mark","q":"mark","i":[{"":"Text","s":"with"},{"":"Space"},{"":"Text","s":"text"}]}]`,
			encoderHTML:  `<a id="mark">with text</a>`,
			encoderSexpr: `((MARK "mark" "mark" "mark" (TEXT "with") (SPACE) (TEXT "text")))`,
			encoderText:  `with text`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Dummy Link",
		zmk:   `[[abc]]`,
		expect: expectMap{
			encoderZJSON: `[{"":"Link","q":"external","s":"abc"}]`,
			encoderHTML:  `<a class="external" href="abc">abc</a>`,
			encoderSexpr: `((LINK-EXTERNAL () "abc"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple URL",
		zmk:   `[[https://zettelstore.de]]`,
		expect: expectMap{
			encoderZJSON: `[{"":"Link","q":"external","s":"https://zettelstore.de"}]`,
			encoderHTML:  `<a class="external" href="https://zettelstore.de">https://zettelstore.de</a>`,
			encoderSexpr: `((LINK-EXTERNAL () "https://zettelstore.de"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "URL with Text",
		zmk:   `[[Home|https://zettelstore.de]]`,
		expect: expectMap{
			encoderZJSON: `[{"":"Link","q":"external","s":"https://zettelstore.de","i":[{"":"Text","s":"Home"}]}]`,
			encoderHTML:  `<a class="external" href="https://zettelstore.de">Home</a>`,
			encoderSexpr: `((LINK-EXTERNAL () "https://zettelstore.de" (TEXT "Home")))`,
			encoderText:  `Home`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple Zettel ID",
		zmk:   `[[00000000000100]]`,
		expect: expectMap{
			encoderZJSON: `[{"":"Link","q":"zettel","s":"00000000000100"}]`,
			encoderHTML:  `<a href="00000000000100">00000000000100</a>`,
			encoderSexpr: `((LINK-ZETTEL () "00000000000100"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Zettel ID with Text",
		zmk:   `[[Config|00000000000100]]`,
		expect: expectMap{
			encoderZJSON: `[{"":"Link","q":"zettel","s":"00000000000100","i":[{"":"Text","s":"Config"}]}]`,
			encoderHTML:  `<a href="00000000000100">Config</a>`,
			encoderSexpr: `((LINK-ZETTEL () "00000000000100" (TEXT "Config")))`,
			encoderText:  `Config`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple Zettel ID with fragment",
		zmk:   `[[00000000000100#frag]]`,
		expect: expectMap{
			encoderZJSON: `[{"":"Link","q":"zettel","s":"00000000000100#frag"}]`,
			encoderHTML:  `<a href="00000000000100#frag">00000000000100#frag</a>`,
			encoderSexpr: `((LINK-ZETTEL () "00000000000100#frag"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Zettel ID with Text and fragment",
		zmk:   `[[Config|00000000000100#frag]]`,
		expect: expectMap{
			encoderZJSON: `[{"":"Link","q":"zettel","s":"00000000000100#frag","i":[{"":"Text","s":"Config"}]}]`,
			encoderHTML:  `<a href="00000000000100#frag">Config</a>`,
			encoderSexpr: `((LINK-ZETTEL () "00000000000100#frag" (TEXT "Config")))`,
			encoderText:  `Config`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Fragment link to self",
		zmk:   `[[#frag]]`,
		expect: expectMap{
			encoderZJSON: `[{"":"Link","q":"self","s":"#frag"}]`,
			encoderHTML:  `<a href="#frag">#frag</a>`,
			encoderSexpr: `((LINK-SELF () "#frag"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Hosted link",
		zmk:   `[[H|/hosted]]`,
		expect: expectMap{
			encoderZJSON: `[{"":"Link","q":"local","s":"/hosted","i":[{"":"Text","s":"H"}]}]`,
			encoderHTML:  `<a href="/hosted">H</a>`,
			encoderSexpr: `((LINK-HOSTED () "/hosted" (TEXT "H")))`,
			encoderText:  `H`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Based link",
		zmk:   `[[B|/based]]`,
		expect: expectMap{
			encoderZJSON: `[{"":"Link","q":"local","s":"/based","i":[{"":"Text","s":"B"}]}]`,
			encoderHTML:  `<a href="/based">B</a>`,
			encoderSexpr: `((LINK-HOSTED () "/based" (TEXT "B")))`,
			encoderText:  `B`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Relative link",
		zmk:   `[[R|../relative]]`,
		expect: expectMap{
			encoderZJSON: `[{"":"Link","q":"local","s":"../relative","i":[{"":"Text","s":"R"}]}]`,
			encoderHTML:  `<a href="../relative">R</a>`,
			encoderSexpr: `((LINK-HOSTED () "../relative" (TEXT "R")))`,
			encoderText:  `R`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Dummy Embed",
		zmk:   `{{abc}}`,
		expect: expectMap{
			encoderZJSON: `[{"":"Embed","s":"abc"}]`,
			encoderHTML:  `<img src="abc">`,
			encoderSexpr: `((EMBED () (EXTERNAL "abc") ""))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "",
		zmk:   ``,
		expect: expectMap{
			encoderZJSON: `[]`,
			encoderHTML:  ``,
			encoderSexpr: `()`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
}
