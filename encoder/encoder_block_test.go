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

var tcsBlock = []zmkTestCase{
	{
		descr: "Empty Zettelmarkup should produce near nothing",
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
		descr: "Simple text: Hello, world",
		zmk:   "Hello, world",
		expect: expectMap{
			encoderZJSON: `[{"":"Para","i":[{"":"Text","s":"Hello,"},{"":"Space"},{"":"Text","s":"world"}]}]`,
			encoderHTML:  "<p>Hello, world</p>",
			encoderSexpr: `([PARA (TEXT "Hello,") (SPACE) (TEXT "world")])`,
			encoderText:  "Hello, world",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple block comment",
		zmk:   "%%%\nNo\nrender\n%%%",
		expect: expectMap{
			encoderZJSON: `[{"":"CommentBlock","s":"No\nrender"}]`,
			encoderHTML:  ``,
			encoderSexpr: `((VERBATIM-COMMENT () "No\nrender"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Rendered block comment",
		zmk:   "%%%{-}\nRender\n%%%",
		expect: expectMap{
			encoderZJSON: `[{"":"CommentBlock","a":{"-":""},"s":"Render"}]`,
			encoderHTML:  "<!--\nRender\n-->",
			encoderSexpr: `((VERBATIM-COMMENT (("-" "")) "Render"))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple Heading",
		zmk:   `=== Top`,
		expect: expectMap{
			encoderZJSON: `[{"":"Heading","n":1,"s":"top","i":[{"":"Text","s":"Top"}]}]`,
			encoderHTML:  "<h2 id=\"top\">Top</h2>",
			encoderSexpr: `([HEADING 1 () "top" "top" (TEXT "Top")])`,
			encoderText:  `Top`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple List",
		zmk:   "* A\n* B\n* C",
		expect: expectMap{
			encoderZJSON: `[{"":"Bullet","c":[[{"":"Para","i":[{"":"Text","s":"A"}]}],[{"":"Para","i":[{"":"Text","s":"B"}]}],[{"":"Para","i":[{"":"Text","s":"C"}]}]]}]`,
			encoderHTML:  "<ul>\n<li>A</li>\n<li>B</li>\n<li>C</li>\n</ul>",
			encoderSexpr: `([UNORDERED ((TEXT "A")) ((TEXT "B")) ((TEXT "C"))])`,
			encoderText:  "A\nB\nC",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Nested List",
		zmk:   "* T1\n** T2\n* T3\n** T4\n** T5\n* T6",
		expect: expectMap{
			encoderZJSON: `[{"":"Bullet","c":[[{"":"Para","i":[{"":"Text","s":"T1"}]},{"":"Bullet","c":[[{"":"Para","i":[{"":"Text","s":"T2"}]}]]}],[{"":"Para","i":[{"":"Text","s":"T3"}]},{"":"Bullet","c":[[{"":"Para","i":[{"":"Text","s":"T4"}]}],[{"":"Para","i":[{"":"Text","s":"T5"}]}]]}],[{"":"Para","i":[{"":"Text","s":"T6"}]}]]}]`,
			encoderHTML: `<ul>
<li><p>T1</p>
<ul>
<li>T2</li>
</ul></li>
<li><p>T3</p>
<ul>
<li>T4</li>
<li>T5</li>
</ul></li>
<li><p>T6</p></li>
</ul>`,
			encoderSexpr: `([UNORDERED ([PARA (TEXT "T1")] (UNORDERED ((TEXT "T2")))) ([PARA (TEXT "T3")] (UNORDERED ((TEXT "T4")) ((TEXT "T5")))) ([PARA (TEXT "T6")])])`,
			encoderText:  "T1\nT2\nT3\nT4\nT5\nT6",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Sequence of two lists",
		zmk:   "* Item1.1\n* Item1.2\n* Item1.3\n\n* Item2.1\n* Item2.2",
		expect: expectMap{
			encoderZJSON: `[{"":"Bullet","c":[[{"":"Para","i":[{"":"Text","s":"Item1.1"}]}],[{"":"Para","i":[{"":"Text","s":"Item1.2"}]}],[{"":"Para","i":[{"":"Text","s":"Item1.3"}]}],[{"":"Para","i":[{"":"Text","s":"Item2.1"}]}],[{"":"Para","i":[{"":"Text","s":"Item2.2"}]}]]}]`,
			encoderHTML:  "<ul>\n<li>Item1.1</li>\n<li>Item1.2</li>\n<li>Item1.3</li>\n<li>Item2.1</li>\n<li>Item2.2</li>\n</ul>",
			encoderSexpr: `([UNORDERED ((TEXT "Item1.1")) ((TEXT "Item1.2")) ((TEXT "Item1.3")) ((TEXT "Item2.1")) ((TEXT "Item2.2"))])`,
			encoderText:  "Item1.1\nItem1.2\nItem1.3\nItem2.1\nItem2.2",
			encoderZmk:   "* Item1.1\n* Item1.2\n* Item1.3\n* Item2.1\n* Item2.2",
		},
	},
	{
		descr: "Simple horizontal rule",
		zmk:   `---`,
		expect: expectMap{
			encoderZJSON: `[{"":"Thematic"}]`,
			encoderHTML:  "<hr>",
			encoderSexpr: `((THEMATIC ()))`,
			encoderText:  ``,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "No list after paragraph",
		zmk:   "Text\n*abc",
		expect: expectMap{
			encoderZJSON: `[{"":"Para","i":[{"":"Text","s":"Text"},{"":"Soft"},{"":"Text","s":"*abc"}]}]`,
			encoderHTML:  "<p>Text\n*abc</p>",
			encoderSexpr: `([PARA (TEXT "Text") (SOFT) (TEXT "*abc")])`,
			encoderText:  `Text *abc`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "A list after paragraph",
		zmk:   "Text\n# abc",
		expect: expectMap{
			encoderZJSON: `[{"":"Para","i":[{"":"Text","s":"Text"}]},{"":"Ordered","c":[[{"":"Para","i":[{"":"Text","s":"abc"}]}]]}]`,
			encoderHTML:  "<p>Text</p>\n<ol>\n<li>abc</li>\n</ol>",
			encoderSexpr: `([PARA (TEXT "Text")] (ORDERED ((TEXT "abc"))))`,
			encoderText:  "Text\nabc",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple Quote Block",
		zmk:   "<<<\nToBeOrNotToBe\n<<< Romeo",
		expect: expectMap{
			encoderZJSON: `[{"":"Excerpt","b":[{"":"Para","i":[{"":"Text","s":"ToBeOrNotToBe"}]}],"i":[{"":"Text","s":"Romeo"}]}]`,
			encoderHTML:  "<blockquote>\n<p>ToBeOrNotToBe</p>\n<cite>Romeo</cite>\n</blockquote>",
			encoderSexpr: `([REGION-QUOTE () ([PARA (TEXT "ToBeOrNotToBe")]) [(TEXT "Romeo")]])`,
			encoderText:  "ToBeOrNotToBe\nRomeo",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Quote Block with multiple paragraphs",
		zmk:   "<<<\nToBeOr\n\nNotToBe\n<<< Romeo",
		expect: expectMap{
			encoderZJSON: `[{"":"Excerpt","b":[{"":"Para","i":[{"":"Text","s":"ToBeOr"}]},{"":"Para","i":[{"":"Text","s":"NotToBe"}]}],"i":[{"":"Text","s":"Romeo"}]}]`,
			encoderHTML:  "<blockquote>\n<p>ToBeOr</p>\n<p>NotToBe</p>\n<cite>Romeo</cite>\n</blockquote>",
			encoderSexpr: `([REGION-QUOTE () ([PARA (TEXT "ToBeOr")] [PARA (TEXT "NotToBe")]) [(TEXT "Romeo")]])`,
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
			encoderZJSON: "[{\"\":\"Poem\",\"b\":[{\"\":\"Para\",\"i\":[{\"\":\"Text\",\"s\":\"A\"},{\"\":\"Space\",\"s\":\"\u00a0\"},{\"\":\"Text\",\"s\":\"line\"},{\"\":\"Hard\"},{\"\":\"Space\",\"s\":\"\u00a0\u00a0\"},{\"\":\"Text\",\"s\":\"another\"},{\"\":\"Space\",\"s\":\"\u00a0\"},{\"\":\"Text\",\"s\":\"line\"},{\"\":\"Hard\"},{\"\":\"Text\",\"s\":\"Back\"}]},{\"\":\"Para\",\"i\":[{\"\":\"Text\",\"s\":\"Paragraph\"}]},{\"\":\"Para\",\"i\":[{\"\":\"Space\",\"s\":\"\u00a0\u00a0\u00a0\u00a0\"},{\"\":\"Text\",\"s\":\"Spacy\"},{\"\":\"Space\",\"s\":\"\u00a0\u00a0\"},{\"\":\"Text\",\"s\":\"Para\"}]}],\"i\":[{\"\":\"Text\",\"s\":\"Author\"}]}]",
			encoderHTML:  "<div>\n<p>A\u00a0line<br>\n\u00a0\u00a0another\u00a0line<br>\nBack</p>\n<p>Paragraph</p>\n<p>\u00a0\u00a0\u00a0\u00a0Spacy\u00a0\u00a0Para</p>\n<cite>Author</cite>\n</div>",
			encoderSexpr: "([REGION-VERSE () ([PARA (TEXT \"A\") (SPACE \"\u00a0\") (TEXT \"line\") (HARD) (SPACE \"\u00a0\u00a0\") (TEXT \"another\") (SPACE \"\u00a0\") (TEXT \"line\") (HARD) (TEXT \"Back\")] [PARA (TEXT \"Paragraph\")] [PARA (SPACE \"\u00a0\u00a0\u00a0\u00a0\") (TEXT \"Spacy\") (SPACE \"\u00a0\u00a0\") (TEXT \"Para\")]) [(TEXT \"Author\")]])",
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
			encoderZJSON: `[{"":"Block","b":[{"":"Para","i":[{"":"Text","s":"A"},{"":"Space"},{"":"Text","s":"simple"},{"":"Soft"},{"":"Space"},{"":"Text","s":"span"},{"":"Soft"},{"":"Text","s":"and"},{"":"Space"},{"":"Text","s":"much"},{"":"Space"},{"":"Text","s":"more"}]}]}]`,
			encoderHTML:  "<div>\n<p>A simple\n span\nand much more</p>\n</div>",
			encoderSexpr: `([REGION-BLOCK () ([PARA (TEXT "A") (SPACE) (TEXT "simple") (SOFT) (SPACE) (TEXT "span") (SOFT) (TEXT "and") (SPACE) (TEXT "much") (SPACE) (TEXT "more")]) []])`,
			encoderText:  `A simple  span and much more`,
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple Verbatim Code",
		zmk:   "```\nHello\nWorld\n```",
		expect: expectMap{
			encoderZJSON: `[{"":"CodeBlock","s":"Hello\nWorld"}]`,
			encoderHTML:  "<pre><code>Hello\nWorld</code></pre>",
			encoderSexpr: `((VERBATIM-CODE () "Hello\nWorld"))`,
			encoderText:  "Hello\nWorld",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple Verbatim Eval",
		zmk:   "~~~\nHello\nWorld\n~~~",
		expect: expectMap{
			encoderZJSON: `[{"":"EvalBlock","s":"Hello\nWorld"}]`,
			encoderHTML:  "<pre><code class=\"zs-eval\">Hello\nWorld</code></pre>",
			encoderSexpr: `((VERBATIM-EVAL () "Hello\nWorld"))`,
			encoderText:  "Hello\nWorld",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple Verbatim Math",
		zmk:   "$$$\nHello\n\\LaTeX\n$$$",
		expect: expectMap{
			encoderZJSON: `[{"":"MathBlock","s":"Hello\n\\LaTeX"}]`,
			encoderHTML:  "<pre><code class=\"zs-math\">Hello\n\\LaTeX</code></pre>",
			encoderSexpr: `((VERBATIM-MATH () "Hello\n\\LaTeX"))`,
			encoderText:  "Hello\n\\LaTeX",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple Description List",
		zmk:   "; Zettel\n: Paper\n: Note\n; Zettelkasten\n: Slip box",
		expect: expectMap{
			encoderZJSON: `[{"":"Description","d":[{"i":[{"":"Text","s":"Zettel"}],"e":[[{"":"Para","i":[{"":"Text","s":"Paper"}]}],[{"":"Para","i":[{"":"Text","s":"Note"}]}]]},{"i":[{"":"Text","s":"Zettelkasten"}],"e":[[{"":"Para","i":[{"":"Text","s":"Slip"},{"":"Space"},{"":"Text","s":"box"}]}]]}]}]`,
			encoderHTML:  "<dl>\n<dt>Zettel</dt>\n<dd>Paper</dd>\n<dd>Note</dd>\n<dt>Zettelkasten</dt>\n<dd>Slip box</dd>\n</dl>",
			encoderSexpr: `([DESCRIPTION [(TEXT "Zettel")] (([PARA (TEXT "Paper")]) ([PARA (TEXT "Note")])) [(TEXT "Zettelkasten")] (([PARA (TEXT "Slip") (SPACE) (TEXT "box")]))])`,
			encoderText:  "Zettel\nPaper\nNote\nZettelkasten\nSlip box",
			encoderZmk:   useZmk,
		},
	},
	{
		descr: "Simple Table",
		zmk:   "|c1|c2|c3\n|d1||d3",
		expect: expectMap{
			encoderZJSON: `[{"":"Table","p":[[],[[{"i":[{"":"Text","s":"c1"}]},{"i":[{"":"Text","s":"c2"}]},{"i":[{"":"Text","s":"c3"}]}],[{"i":[{"":"Text","s":"d1"}]},{"i":[]},{"i":[{"":"Text","s":"d3"}]}]]]}]`,
			encoderHTML: `<table>
<tbody>
<tr><td>c1</td><td>c2</td><td>c3</td></tr>
<tr><td>d1</td><td></td><td>d3</td></tr>
</tbody>
</table>`,
			encoderSexpr: `([TABLE () ([CELL (TEXT "c1")] [CELL (TEXT "c2")] [CELL (TEXT "c3")]) ([CELL (TEXT "d1")] [CELL] [CELL (TEXT "d3")])])`,
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
			encoderZJSON: `[{"":"Table","p":[[{"s":">","i":[{"":"Text","s":"h1"}]},{"i":[{"":"Text","s":"h2"}]},{"s":":","i":[{"":"Text","s":"h3"}]}],[[{"s":"<","i":[{"":"Text","s":"c1"}]},{"i":[{"":"Text","s":"c2"}]},{"s":":","i":[{"":"Text","s":"c3"}]}],[{"s":">","i":[{"":"Text","s":"f1"}]},{"i":[{"":"Text","s":"f2"}]},{"s":":","i":[{"":"Text","s":"=f3"}]}]]]}]`,
			encoderHTML: `<table>
<thead>
<tr><th class="right">h1</th><th>h2</th><th class="center">h3</th></tr>
</thead>
<tbody>
<tr><td class="left">c1</td><td>c2</td><td class="center">c3</td></tr>
<tr><td class="right">f1</td><td>f2</td><td class="center">=f3</td></tr>
</tbody>
</table>`,
			encoderSexpr: `([TABLE ([CELL-RIGHT (TEXT "h1")] [CELL (TEXT "h2")] [CELL-CENTER (TEXT "h3")]) ([CELL-LEFT (TEXT "c1")] [CELL (TEXT "c2")] [CELL-CENTER (TEXT "c3")]) ([CELL-RIGHT (TEXT "f1")] [CELL (TEXT "f2")] [CELL-CENTER (TEXT "=f3")])])`,
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
			encoderZJSON: `[{"":"Para","i":[{"":"Text","s":"Text"},{"":"Footnote","i":[{"":"Text","s":"Footnote"}]}]}]`,
			encoderHTML: `<p>Text<sup id="fnref:1"><a class="zs-noteref" href="#fn:1" role="doc-noteref">1</a></sup></p>
<ol class="zs-endnotes">
<li class="zs-endnote" id="fn:1" role="doc-endnote" value="1">Footnote <a class="zs-endnote-backref" href="#fnref:1" role="doc-backlink">&#x21a9;&#xfe0e;</a></li>
</ol>
`,
			encoderSexpr: `([PARA (TEXT "Text") [FOOTNOTE () (TEXT "Footnote")]])`,
			encoderText:  "Text Footnote",
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
			encoderText:  "",
			encoderZmk:   useZmk,
		},
	},
}

// func TestEncoderBlock(t *testing.T) {
// 	executeTestCases(t, tcsBlock)
// }
