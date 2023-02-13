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

var tcsBlock = []zmkTestCase{
	{
		descr: "Empty Zettelmarkup should produce near nothing",
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
		descr: "Simple text: Hello, world",
		zmk:   "Hello, world",
		expect: expectMap{
			encoderHTML:  "<p>Hello, world</p>",
			encoderMD:    "Hello, world",
			encoderSexpr: `((PARA (TEXT "Hello,") (SPACE) (TEXT "world")))`,
			encoderText:  "Hello, world",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple block comment",
		zmk:   "%%%\nNo\nrender\n%%%",
		expect: expectMap{
			encoderHTML:  ``,
			encoderMD:    "",
			encoderSexpr: `((VERBATIM-COMMENT () "No\nrender"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Rendered block comment",
		zmk:   "%%%{-}\nRender\n%%%",
		expect: expectMap{
			encoderHTML:  "<!--\nRender\n-->",
			encoderMD:    "",
			encoderSexpr: `((VERBATIM-COMMENT (("-" "")) "Render"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple Heading",
		zmk:   `=== Top`,
		expect: expectMap{
			encoderHTML:  "<h2 id=\"top\">Top</h2>",
			encoderMD:    "# Top",
			encoderSexpr: `((HEADING 1 () "top" "top" (TEXT "Top")))`,
			encoderText:  `Top`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple List",
		zmk:   "* A\n* B\n* C",
		expect: expectMap{
			encoderHTML:  "<ul><li>A</li><li>B</li><li>C</li></ul>",
			encoderMD:    "* A\n* B\n* C",
			encoderSexpr: `((UNORDERED ((TEXT "A")) ((TEXT "B")) ((TEXT "C"))))`,
			encoderText:  "A\nB\nC",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Nested List",
		zmk:   "* T1\n** T2\n* T3\n** T4\n** T5\n* T6",
		expect: expectMap{
			encoderHTML:  `<ul><li><p>T1</p><ul><li>T2</li></ul></li><li><p>T3</p><ul><li>T4</li><li>T5</li></ul></li><li><p>T6</p></li></ul>`,
			encoderMD:    "* T1\n    * T2\n* T3\n    * T4\n    * T5\n* T6",
			encoderSexpr: `((UNORDERED ((PARA (TEXT "T1")) (UNORDERED ((TEXT "T2")))) ((PARA (TEXT "T3")) (UNORDERED ((TEXT "T4")) ((TEXT "T5")))) ((PARA (TEXT "T6")))))`,
			encoderText:  "T1\nT2\nT3\nT4\nT5\nT6",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Sequence of two lists",
		zmk:   "* Item1.1\n* Item1.2\n* Item1.3\n\n* Item2.1\n* Item2.2",
		expect: expectMap{
			encoderHTML:  "<ul><li>Item1.1</li><li>Item1.2</li><li>Item1.3</li><li>Item2.1</li><li>Item2.2</li></ul>",
			encoderMD:    "* Item1.1\n* Item1.2\n* Item1.3\n* Item2.1\n* Item2.2",
			encoderSexpr: `((UNORDERED ((TEXT "Item1.1")) ((TEXT "Item1.2")) ((TEXT "Item1.3")) ((TEXT "Item2.1")) ((TEXT "Item2.2"))))`,
			encoderText:  "Item1.1\nItem1.2\nItem1.3\nItem2.1\nItem2.2",
			encoderZmk:   "* Item1.1\n* Item1.2\n* Item1.3\n* Item2.1\n* Item2.2",
		},
	},
	{
		descr: "Simple horizontal rule",
		zmk:   `---`,
		expect: expectMap{
			encoderHTML:  "<hr>",
			encoderMD:    "---",
			encoderSexpr: `((THEMATIC ()))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "No list after paragraph",
		zmk:   "Text\n*abc",
		expect: expectMap{
			encoderHTML:  "<p>Text *abc</p>",
			encoderMD:    "Text\n*abc",
			encoderSexpr: `((PARA (TEXT "Text") (SOFT) (TEXT "*abc")))`,
			encoderText:  `Text *abc`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "A list after paragraph",
		zmk:   "Text\n# abc",
		expect: expectMap{
			encoderHTML:  "<p>Text</p><ol><li>abc</li></ol>",
			encoderMD:    "Text\n\n1. abc",
			encoderSexpr: `((PARA (TEXT "Text")) (ORDERED ((TEXT "abc"))))`,
			encoderText:  "Text\nabc",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple List Quote",
		zmk:   "> ToBeOrNotToBe",
		expect: expectMap{
			encoderHTML:  "<blockquote><p>ToBeOrNotToBe</p></blockquote>",
			encoderMD:    "> ToBeOrNotToBe",
			encoderSexpr: `((QUOTATION ((TEXT "ToBeOrNotToBe"))))`,
			encoderText:  "ToBeOrNotToBe",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple Quote Block",
		zmk:   "<<<\nToBeOrNotToBe\n<<< Romeo",
		expect: expectMap{
			encoderHTML:  "<blockquote><p>ToBeOrNotToBe</p><cite>Romeo</cite></blockquote>",
			encoderMD:    "> ToBeOrNotToBe",
			encoderSexpr: `((REGION-QUOTE () ((PARA (TEXT "ToBeOrNotToBe"))) ((TEXT "Romeo"))))`,
			encoderText:  "ToBeOrNotToBe\nRomeo",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Quote Block with multiple paragraphs",
		zmk:   "<<<\nToBeOr\n\nNotToBe\n<<< Romeo",
		expect: expectMap{
			encoderHTML:  "<blockquote><p>ToBeOr</p><p>NotToBe</p><cite>Romeo</cite></blockquote>",
			encoderMD:    "> ToBeOr\n\n> NotToBe",
			encoderSexpr: `((REGION-QUOTE () ((PARA (TEXT "ToBeOr")) (PARA (TEXT "NotToBe"))) ((TEXT "Romeo"))))`,
			encoderText:  "ToBeOr\nNotToBe\nRomeo",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Verse block",
		zmk: `"""
A line
  another line
Back

Paragraph

    Spacy  Para
""" Author`,
		expect: expectMap{
			encoderHTML:  "<div><p>A\u00a0line<br>\u00a0\u00a0another\u00a0line<br>Back</p><p>Paragraph</p><p>\u00a0\u00a0\u00a0\u00a0Spacy\u00a0\u00a0Para</p><cite>Author</cite></div>",
			encoderMD:    "",
			encoderSexpr: "((REGION-VERSE () ((PARA (TEXT \"A\") (SPACE \"\u00a0\") (TEXT \"line\") (HARD) (SPACE \"\u00a0\u00a0\") (TEXT \"another\") (SPACE \"\u00a0\") (TEXT \"line\") (HARD) (TEXT \"Back\")) (PARA (TEXT \"Paragraph\")) (PARA (SPACE \"\u00a0\u00a0\u00a0\u00a0\") (TEXT \"Spacy\") (SPACE \"\u00a0\u00a0\") (TEXT \"Para\"))) ((TEXT \"Author\"))))",
			encoderText:  "A line\n another line\nBack\nParagraph\n Spacy Para\nAuthor",
			encoderZmk:   "\"\"\"\nA\u00a0line\\\n\u00a0\u00a0another\u00a0line\\\nBack\nParagraph\n\u00a0\u00a0\u00a0\u00a0Spacy\u00a0\u00a0Para\n\"\"\" Author",
		},
	},
	{
		descr: "Span Block",
		zmk: `:::
A simple
   span
and much more
:::`,
		expect: expectMap{
			encoderHTML:  "<div><p>A simple  span and much more</p></div>",
			encoderMD:    "",
			encoderSexpr: `((REGION-BLOCK () ((PARA (TEXT "A") (SPACE) (TEXT "simple") (SOFT) (SPACE) (TEXT "span") (SOFT) (TEXT "and") (SPACE) (TEXT "much") (SPACE) (TEXT "more"))) ()))`,
			encoderText:  `A simple  span and much more`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple Verbatim Code",
		zmk:   "```\nHello\nWorld\n```",
		expect: expectMap{
			encoderHTML:  "<pre><code>Hello\nWorld</code></pre>",
			encoderMD:    "    Hello\n    World",
			encoderSexpr: `((VERBATIM-CODE () "Hello\nWorld"))`,
			encoderText:  "Hello\nWorld",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple Verbatim Code with visible spaces",
		zmk:   "```{-}\nHello World\n```",
		expect: expectMap{
			encoderHTML:  "<pre><code>Hello\u2423World</code></pre>",
			encoderMD:    "    Hello World",
			encoderSexpr: `((VERBATIM-CODE (("-" "")) "Hello World"))`,
			encoderText:  "Hello World",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple Verbatim Eval",
		zmk:   "~~~\nHello\nWorld\n~~~",
		expect: expectMap{
			encoderHTML:  "<pre><code class=\"zs-eval\">Hello\nWorld</code></pre>",
			encoderMD:    "",
			encoderSexpr: `((VERBATIM-EVAL () "Hello\nWorld"))`,
			encoderText:  "Hello\nWorld",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple Verbatim Math",
		zmk:   "$$$\nHello\n\\LaTeX\n$$$",
		expect: expectMap{
			encoderHTML:  "<pre><code class=\"zs-math\">Hello\n\\LaTeX</code></pre>",
			encoderMD:    "",
			encoderSexpr: `((VERBATIM-MATH () "Hello\n\\LaTeX"))`,
			encoderText:  "Hello\n\\LaTeX",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple Description List",
		zmk:   "; Zettel\n: Paper\n: Note\n; Zettelkasten\n: Slip box",
		expect: expectMap{
			encoderHTML:  "<dl><dt>Zettel</dt><dd>Paper</dd><dd>Note</dd><dt>Zettelkasten</dt><dd>Slip box</dd></dl>",
			encoderMD:    "",
			encoderSexpr: `((DESCRIPTION ((TEXT "Zettel")) (((TEXT "Paper")) ((TEXT "Note"))) ((TEXT "Zettelkasten")) (((TEXT "Slip") (SPACE) (TEXT "box")))))`,
			encoderText:  "Zettel\nPaper\nNote\nZettelkasten\nSlip box",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple Table",
		zmk:   "|c1|c2|c3\n|d1||d3",
		expect: expectMap{
			encoderHTML:  `<table><tbody><tr><td>c1</td><td>c2</td><td>c3</td></tr><tr><td>d1</td><td></td><td>d3</td></tr></tbody></table>`,
			encoderMD:    "",
			encoderSexpr: `((TABLE () ((CELL (TEXT "c1")) (CELL (TEXT "c2")) (CELL (TEXT "c3"))) ((CELL (TEXT "d1")) (CELL) (CELL (TEXT "d3")))))`,
			encoderText:  "c1 c2 c3\nd1  d3",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Table with alignment and comment",
		zmk: `|h1>|=h2|h3:|
|%--+---+---+
|<c1|c2|:c3|
|f1|f2|=f3`,
		expect: expectMap{
			encoderHTML:  `<table><thead><tr><td class="right">h1</td><td>h2</td><td class="center">h3</td></tr></thead><tbody><tr><td class="left">c1</td><td>c2</td><td class="center">c3</td></tr><tr><td class="right">f1</td><td>f2</td><td class="center">=f3</td></tr></tbody></table>`,
			encoderMD:    "",
			encoderSexpr: `((TABLE ((CELL-RIGHT (TEXT "h1")) (CELL (TEXT "h2")) (CELL-CENTER (TEXT "h3"))) ((CELL-LEFT (TEXT "c1")) (CELL (TEXT "c2")) (CELL-CENTER (TEXT "c3"))) ((CELL-RIGHT (TEXT "f1")) (CELL (TEXT "f2")) (CELL-CENTER (TEXT "=f3")))))`,
			encoderText:  "h1 h2 h3\nc1 c2 c3\nf1 f2 =f3",
			encoderZmk: `|=h1>|=h2|=h3:
|<c1|c2|c3
|f1|f2|=f3`,
		},
	},
	{
		descr: "Simple Endnotes",
		zmk:   `Text[^Footnote]`,
		expect: expectMap{
			encoderHTML:  `<p>Text<sup id="fnref:1"><a class="zs-noteref" href="#fn:1" role="doc-noteref">1</a></sup></p><ol class="zs-endnotes"><li class="zs-endnote" id="fn:1" role="doc-endnote" value="1">Footnote <a class="zs-endnote-backref" href="#fnref:1" role="doc-backlink">&#x21a9;&#xfe0e;</a></li></ol>`,
			encoderMD:    "Text",
			encoderSexpr: `((PARA (TEXT "Text") (FOOTNOTE () (TEXT "Footnote"))))`,
			encoderText:  "Text Footnote",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Transclusion",
		zmk:   `{{{http://example.com/image}}}{width="100px"}`,
		expect: expectMap{
			encoderHTML:  `<p><img class="external" src="http://example.com/image" width="100px"></p>`,
			encoderMD:    "",
			encoderSexpr: `((TRANSCLUDE (("width" "100px")) (EXTERNAL "http://example.com/image")))`,
			encoderText:  "",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "A paragraph with a inline comment only should be empty in HTML",
		zmk:   `%% Comment`,
		expect: expectMap{
			encoderHTML:  ``,
			encoderSexpr: `((PARA (LITERAL-COMMENT () "Comment")))`,
			encoderText:  "",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "",
		zmk:   ``,
		expect: expectMap{
			encoderHTML:  ``,
			encoderSexpr: `()`,
			encoderSHTML: `()`,
			encoderText:  "",
			encoderZmk:   useZmk,
		},
	},
}

// func TestEncoderBlock(t *testing.T) {
// 	executeTestCases(t, tcsBlock)
// }
