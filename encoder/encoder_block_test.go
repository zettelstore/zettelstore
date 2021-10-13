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

import "testing"

var tcsBlock = []zmkTestCase{
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
		descr: "Simple text: Hello, world",
		zmk:   "Hello, world",
		expect: expectMap{
			encoderDJSON:  `[{"t":"Para","i":[{"t":"Text","s":"Hello,"},{"t":"Space"},{"t":"Text","s":"world"}]}]`,
			encoderHTML:   "<p>Hello, world</p>",
			encoderNative: `[Para Text "Hello,",Space,Text "world"]`,
			encoderText:   "Hello, world",
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Simple block comment",
		zmk:   "%%%\nNo\nrender\n%%%",
		expect: expectMap{
			encoderDJSON:  `[{"t":"CommentBlock","l":["No","render"]}]`,
			encoderHTML:   ``,
			encoderNative: `[CommentBlock "No\nrender"]`,
			encoderText:   ``,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Rendered block comment",
		zmk:   "%%%{-}\nRender\n%%%",
		expect: expectMap{
			encoderDJSON:  `[{"t":"CommentBlock","a":{"-":""},"l":["Render"]}]`,
			encoderHTML:   "<!--\nRender\n-->",
			encoderNative: `[CommentBlock ("",[-]) "Render"]`,
			encoderText:   ``,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Simple Heading",
		zmk:   `=== Top`,
		expect: expectMap{
			// TODO: 2-->1
			encoderDJSON:  `[{"t":"Heading","n":2,"s":"top","i":[{"t":"Text","s":"Top"}]}]`,
			encoderHTML:   "<h2 id=\"top\">Top</h2>",
			encoderNative: `[Heading 2 #top Text "Top"]`, // TODO: 2 --> 1
			encoderText:   `Top`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Einfache Liste",
		zmk:   "* A\n* B\n* C",
		expect: expectMap{
			encoderDJSON: `[{"t":"BulletList","c":[[{"t":"Para","i":[{"t":"Text","s":"A"}]}],[{"t":"Para","i":[{"t":"Text","s":"B"}]}],[{"t":"Para","i":[{"t":"Text","s":"C"}]}]]}]`,
			encoderHTML:  "<ul>\n<li>A</li>\n<li>B</li>\n<li>C</li>\n</ul>",
			encoderNative: `[BulletList
 [[Para Text "A"]],
 [[Para Text "B"]],
 [[Para Text "C"]]]`,
			encoderText: "A\nB\nC",
			encoderZmk:  useZmk,
		},
	},
	{
		descr: "Schachtelliste",
		zmk:   "* T1\n** T2\n* T3\n** T4\n* T5",
		expect: expectMap{
			encoderDJSON: `[{"t":"BulletList","c":[[{"t":"Para","i":[{"t":"Text","s":"T1"}]},{"t":"BulletList","c":[[{"t":"Para","i":[{"t":"Text","s":"T2"}]}]]}],[{"t":"Para","i":[{"t":"Text","s":"T3"}]},{"t":"BulletList","c":[[{"t":"Para","i":[{"t":"Text","s":"T4"}]}]]}],[{"t":"Para","i":[{"t":"Text","s":"T5"}]}]]}]`,
			encoderHTML: `<ul>
<li>
<p>T1</p>
<ul>
<li>T2</li>
</ul>
</li>
<li>
<p>T3</p>
<ul>
<li>T4</li>
</ul>
</li>
<li>
<p>T5</p>
</li>
</ul>`,
			encoderNative: `[BulletList
 [[Para Text "T1"],
  [BulletList
   [[Para Text "T2"]]]],
 [[Para Text "T3"],
  [BulletList
   [[Para Text "T4"]]]],
 [[Para Text "T5"]]]`,
			encoderText: "T1\nT2\nT3\nT4\nT5",
			encoderZmk:  useZmk,
		},
	},
	{
		descr: "Zwei Listen hintereinander",
		zmk:   "* Item1.1\n* Item1.2\n* Item1.3\n\n* Item2.1\n* Item2.2",
		expect: expectMap{
			encoderDJSON: `[{"t":"BulletList","c":[[{"t":"Para","i":[{"t":"Text","s":"Item1.1"}]}],[{"t":"Para","i":[{"t":"Text","s":"Item1.2"}]}],[{"t":"Para","i":[{"t":"Text","s":"Item1.3"}]}],[{"t":"Para","i":[{"t":"Text","s":"Item2.1"}]}],[{"t":"Para","i":[{"t":"Text","s":"Item2.2"}]}]]}]`,
			encoderHTML:  "<ul>\n<li>Item1.1</li>\n<li>Item1.2</li>\n<li>Item1.3</li>\n<li>Item2.1</li>\n<li>Item2.2</li>\n</ul>",
			encoderNative: `[BulletList
 [[Para Text "Item1.1"]],
 [[Para Text "Item1.2"]],
 [[Para Text "Item1.3"]],
 [[Para Text "Item2.1"]],
 [[Para Text "Item2.2"]]]`,
			encoderText: "Item1.1\nItem1.2\nItem1.3\nItem2.1\nItem2.2",
			encoderZmk:  "* Item1.1\n* Item1.2\n* Item1.3\n* Item2.1\n* Item2.2",
		},
	},
	{
		descr: "Simple horizontal rule",
		zmk:   `---`,
		expect: expectMap{
			encoderDJSON:  `[{"t":"Hrule"}]`,
			encoderHTML:   "<hr>",
			encoderNative: `[Hrule]`,
			encoderText:   ``,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "No list after paragraph",
		zmk:   "Text\n*abc",
		expect: expectMap{
			encoderDJSON:  `[{"t":"Para","i":[{"t":"Text","s":"Text"},{"t":"Soft"},{"t":"Text","s":"*abc"}]}]`,
			encoderHTML:   "<p>Text\n*abc</p>",
			encoderNative: `[Para Text "Text",Space,Text "*abc"]`,
			encoderText:   `Text *abc`,
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "A list after paragraph",
		zmk:   "Text\n* abc",
		expect: expectMap{
			encoderDJSON: `[{"t":"Para","i":[{"t":"Text","s":"Text"}]},{"t":"BulletList","c":[[{"t":"Para","i":[{"t":"Text","s":"abc"}]}]]}]`,
			encoderHTML:  "<p>Text</p>\n<ul>\n<li>abc</li>\n</ul>",
			encoderNative: `[Para Text "Text"],
[BulletList
 [[Para Text "abc"]]]`,
			encoderText: "Text\nabc",
			encoderZmk:  useZmk,
		},
	},
	{
		descr: "Simple Quote Block",
		zmk:   "<<<\nToBeOrNotToBe\n<<< Romeo",
		expect: expectMap{
			encoderDJSON: `[{"t":"QuoteBlock","b":[{"t":"Para","i":[{"t":"Text","s":"ToBeOrNotToBe"}]}],"i":[{"t":"Text","s":"Romeo"}]}]`,
			encoderHTML:  "<blockquote>\n<p>ToBeOrNotToBe</p>\n<cite>Romeo</cite>\n</blockquote>",
			encoderNative: `[QuoteBlock
 [[Para Text "ToBeOrNotToBe"]],
 [Cite Text "Romeo"]]`,
			encoderText: "ToBeOrNotToBe\nRomeo",
			encoderZmk:  useZmk,
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
			encoderDJSON:  "[{\"t\":\"VerseBlock\",\"b\":[{\"t\":\"Para\",\"i\":[{\"t\":\"Text\",\"s\":\"A\u00a0line\"},{\"t\":\"Hard\"},{\"t\":\"Text\",\"s\":\"\u00a0\u00a0another\u00a0line\"},{\"t\":\"Hard\"},{\"t\":\"Text\",\"s\":\"Back\"}]},{\"t\":\"Para\",\"i\":[{\"t\":\"Text\",\"s\":\"Paragraph\"}]},{\"t\":\"Para\",\"i\":[{\"t\":\"Text\",\"s\":\"\u00a0\u00a0\u00a0\u00a0Spacy\u00a0\u00a0Para\"}]}],\"i\":[{\"t\":\"Text\",\"s\":\"Author\"}]}]",
			encoderHTML:   "<div>\n<p>A\u00a0line<br>\n\u00a0\u00a0another\u00a0line<br>\nBack</p>\n<p>Paragraph</p>\n<p>\u00a0\u00a0\u00a0\u00a0Spacy\u00a0\u00a0Para</p>\n<cite>Author</cite>\n</div>",
			encoderNative: "[VerseBlock\n [[Para Text \"A\u00a0line\",Break,Text \"\u00a0\u00a0another\u00a0line\",Break,Text \"Back\"],\n  [Para Text \"Paragraph\"],\n  [Para Text \"\u00a0\u00a0\u00a0\u00a0Spacy\u00a0\u00a0Para\"]],\n [Cite Text \"Author\"]]",
			encoderText:   "A\u00a0line\n\u00a0\u00a0another\u00a0line\nBack\nParagraph\n\u00a0\u00a0\u00a0\u00a0Spacy\u00a0\u00a0Para\nAuthor",
			encoderZmk:    "\"\"\"\nA\u00a0line\\\n\u00a0\u00a0another\u00a0line\\\nBack\nParagraph\n\u00a0\u00a0\u00a0\u00a0Spacy\u00a0\u00a0Para\n\"\"\" Author",
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
			encoderDJSON: `[{"t":"SpanBlock","b":[{"t":"Para","i":[{"t":"Text","s":"A"},{"t":"Space"},{"t":"Text","s":"simple"},{"t":"Soft"},{"t":"Space","n":3},{"t":"Text","s":"span"},{"t":"Soft"},{"t":"Text","s":"and"},{"t":"Space"},{"t":"Text","s":"much"},{"t":"Space"},{"t":"Text","s":"more"}]}]}]`,
			encoderHTML:  "<div>\n<p>A simple\n span\nand much more</p>\n</div>",
			encoderNative: `[SpanBlock
 [[Para Text "A",Space,Text "simple",Space,Space 3,Text "span",Space,Text "and",Space,Text "much",Space,Text "more"]]]`,
			encoderText: `A simple  span and much more`,
			encoderZmk:  useZmk,
		},
	},
	{
		descr: "Simple Verbatim",
		zmk:   "```\nHello\nWorld\n```",
		expect: expectMap{
			encoderDJSON:  `[{"t":"CodeBlock","l":["Hello","World"]}]`,
			encoderHTML:   "<pre><code>Hello\nWorld\n</code></pre>",
			encoderNative: `[CodeBlock "Hello\nWorld"]`,
			encoderText:   "Hello\nWorld",
			encoderZmk:    useZmk,
		},
	},
	{
		descr: "Simple Description List",
		zmk:   "; Zettel\n: Paper\n: Note\n; Zettelkasten\n: Slip box",
		expect: expectMap{
			encoderDJSON: `[{"t":"DescriptionList","g":[[[{"t":"Text","s":"Zettel"}],[{"t":"Para","i":[{"t":"Text","s":"Paper"}]}],[{"t":"Para","i":[{"t":"Text","s":"Note"}]}]],[[{"t":"Text","s":"Zettelkasten"}],[{"t":"Para","i":[{"t":"Text","s":"Slip"},{"t":"Space"},{"t":"Text","s":"box"}]}]]]}]`,
			encoderHTML:  "<dl>\n<dt>Zettel</dt>\n<dd>Paper</dd>\n<dd>Note</dd>\n<dt>Zettelkasten</dt>\n<dd>Slip box</dd>\n</dl>",
			encoderNative: `[DescriptionList
 [Term [Text "Zettel"],
  [Description
   [Para Text "Paper"]],
  [Description
   [Para Text "Note"]]],
 [Term [Text "Zettelkasten"],
  [Description
   [Para Text "Slip",Space,Text "box"]]]]`,
			encoderText: "Zettel\nPaper\nNote\nZettelkasten\nSlip box",
			encoderZmk:  useZmk,
		},
	},
	{
		descr: "Simple Table",
		zmk:   "|c1|c2|c3\n|d1||d3",
		expect: expectMap{
			encoderDJSON: `[{"t":"Table","p":[[],[[["",[{"t":"Text","s":"c1"}]],["",[{"t":"Text","s":"c2"}]],["",[{"t":"Text","s":"c3"}]]],[["",[{"t":"Text","s":"d1"}]],["",[]],["",[{"t":"Text","s":"d3"}]]]]]}]`,
			encoderHTML: `<table>
<tbody>
<tr><td>c1</td><td>c2</td><td>c3</td></tr>
<tr><td>d1</td><td></td><td>d3</td></tr>
</tbody>
</table>`,
			encoderNative: `[Table
 [Row [Cell Default Text "c1"],[Cell Default Text "c2"],[Cell Default Text "c3"]],
 [Row [Cell Default Text "d1"],[Cell Default],[Cell Default Text "d3"]]]`,
			encoderText: "c1 c2 c3\nd1  d3",
			encoderZmk:  useZmk,
		},
	},
	{
		descr: "Table with alignment and comment",
		zmk: `|h1>|=h2|h3:|
|%--+---+---+
|<c1|c2|:c3|
|f1|f2|=f3`,
		expect: expectMap{
			encoderDJSON: `[{"t":"Table","p":[[[">",[{"t":"Text","s":"h1"}]],["",[{"t":"Text","s":"h2"}]],[":",[{"t":"Text","s":"h3"}]]],[[["<",[{"t":"Text","s":"c1"}]],["",[{"t":"Text","s":"c2"}]],[":",[{"t":"Text","s":"c3"}]]],[[">",[{"t":"Text","s":"f1"}]],["",[{"t":"Text","s":"f2"}]],[":",[{"t":"Text","s":"=f3"}]]]]]}]`,
			encoderHTML: `<table>
<thead>
<tr><th style="text-align:right">h1</th><th>h2</th><th style="text-align:center">h3</th></tr>
</thead>
<tbody>
<tr><td style="text-align:left">c1</td><td>c2</td><td style="text-align:center">c3</td></tr>
<tr><td style="text-align:right">f1</td><td>f2</td><td style="text-align:center">=f3</td></tr>
</tbody>
</table>`,
			encoderNative: `[Table
 [Header [Cell Right Text "h1"],[Cell Default Text "h2"],[Cell Center Text "h3"]],
 [Row [Cell Left Text "c1"],[Cell Default Text "c2"],[Cell Center Text "c3"]],
 [Row [Cell Right Text "f1"],[Cell Default Text "f2"],[Cell Center Text "=f3"]]]`,
			encoderText: "h1 h2 h3\nc1 c2 c3\nf1 f2 =f3",
			encoderZmk: `|=h1>|=h2|=h3:
|<c1|c2|c3
|f1|f2|=f3`,
		},
	},
	{
		descr: "",
		zmk:   ``,
		expect: expectMap{
			encoderDJSON:  `[]`,
			encoderHTML:   ``,
			encoderNative: ``,
			encoderText:   "",
			encoderZmk:    useZmk,
		},
	},
}

func TestEncoderBlock(t *testing.T) {
	executeTestCases(t, tcsBlock)
}
