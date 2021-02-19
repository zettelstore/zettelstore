//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package jsonenc encodes the abstract syntax tree into JSON.
package jsonenc

import (
	"fmt"
	"io"
	"sort"
	"strconv"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
)

func init() {
	encoder.Register("djson", encoder.Info{
		Create: func() encoder.Encoder { return &jsonDetailEncoder{} },
	})
}

type jsonDetailEncoder struct {
	adaptLink  func(*ast.LinkNode) ast.InlineNode
	adaptImage func(*ast.ImageNode) ast.InlineNode
	title      ast.InlineSlice
}

// SetOption sets an option for the encoder
func (je *jsonDetailEncoder) SetOption(option encoder.Option) {
	switch opt := option.(type) {
	case *encoder.TitleOption:
		je.title = opt.Inline
	case *encoder.AdaptLinkOption:
		je.adaptLink = opt.Adapter
	case *encoder.AdaptImageOption:
		je.adaptImage = opt.Adapter
	}
}

// WriteZettel writes the encoded zettel to the writer.
func (je *jsonDetailEncoder) WriteZettel(
	w io.Writer, zn *ast.ZettelNode, inhMeta bool) (int, error) {
	v := newDetailVisitor(w, je)
	v.b.WriteString("{\"meta\":{\"title\":")
	v.acceptInlineSlice(zn.Title)
	if inhMeta {
		v.writeMeta(zn.InhMeta, false)
	} else {
		v.writeMeta(zn.Zettel.Meta, false)
	}
	v.b.WriteByte('}')
	v.b.WriteString(",\"content\":")
	v.acceptBlockSlice(zn.Ast)
	v.b.WriteByte('}')
	length, err := v.b.Flush()
	return length, err
}

// WriteMeta encodes meta data as JSON.
func (je *jsonDetailEncoder) WriteMeta(w io.Writer, m *meta.Meta) (int, error) {
	v := newDetailVisitor(w, je)
	v.b.WriteByte('{')
	if je.title == nil {
		v.writeMeta(m, true)
	} else {
		v.b.WriteString("\"title\":")
		v.acceptInlineSlice(je.title)
		v.writeMeta(m, false)
	}
	v.b.WriteByte('}')
	length, err := v.b.Flush()
	return length, err
}

func (je *jsonDetailEncoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return je.WriteBlocks(w, zn.Ast)
}

// WriteBlocks writes a block slice to the writer
func (je *jsonDetailEncoder) WriteBlocks(w io.Writer, bs ast.BlockSlice) (int, error) {
	v := newDetailVisitor(w, je)
	v.acceptBlockSlice(bs)
	length, err := v.b.Flush()
	return length, err
}

// WriteInlines writes an inline slice to the writer
func (je *jsonDetailEncoder) WriteInlines(w io.Writer, is ast.InlineSlice) (int, error) {
	v := newDetailVisitor(w, je)
	v.acceptInlineSlice(is)
	length, err := v.b.Flush()
	return length, err
}

// detailVisitor writes the abstract syntax tree to an io.Writer.
type detailVisitor struct {
	b   encoder.BufWriter
	enc *jsonDetailEncoder
}

func newDetailVisitor(w io.Writer, je *jsonDetailEncoder) *detailVisitor {
	return &detailVisitor{b: encoder.NewBufWriter(w), enc: je}
}

// VisitPara emits JSON code for a paragraph.
func (v *detailVisitor) VisitPara(pn *ast.ParaNode) {
	v.writeNodeStart("Para")
	v.writeContentStart('i')
	v.acceptInlineSlice(pn.Inlines)
	v.b.WriteByte('}')
}

var verbatimCode = map[ast.VerbatimCode]string{
	ast.VerbatimProg:    "CodeBlock",
	ast.VerbatimComment: "CommentBlock",
	ast.VerbatimHTML:    "HTMLBlock",
}

// VisitVerbatim emits JSON code for verbatim lines.
func (v *detailVisitor) VisitVerbatim(vn *ast.VerbatimNode) {
	code, ok := verbatimCode[vn.Code]
	if !ok {
		panic(fmt.Sprintf("Unknown verbatim code %v", vn.Code))
	}
	v.writeNodeStart(code)
	v.visitAttributes(vn.Attrs)
	v.writeContentStart('l')
	for i, line := range vn.Lines {
		if i > 0 {
			v.b.WriteByte(',')
		}
		writeEscaped(&v.b, line)
	}
	v.b.WriteString("]}")
}

var regionCode = map[ast.RegionCode]string{
	ast.RegionSpan:  "SpanBlock",
	ast.RegionQuote: "QuoteBlock",
	ast.RegionVerse: "VerseBlock",
}

// VisitRegion writes JSON code for block regions.
func (v *detailVisitor) VisitRegion(rn *ast.RegionNode) {
	code, ok := regionCode[rn.Code]
	if !ok {
		panic(fmt.Sprintf("Unknown region code %v", rn.Code))
	}
	v.writeNodeStart(code)
	v.visitAttributes(rn.Attrs)
	v.writeContentStart('b')
	v.acceptBlockSlice(rn.Blocks)
	if len(rn.Inlines) > 0 {
		v.writeContentStart('i')
		v.acceptInlineSlice(rn.Inlines)
	}
	v.b.WriteByte('}')
}

// VisitHeading writes the JSON code for a heading.
func (v *detailVisitor) VisitHeading(hn *ast.HeadingNode) {
	v.writeNodeStart("Heading")
	v.visitAttributes(hn.Attrs)
	v.writeContentStart('n')
	v.b.WriteString(strconv.Itoa(hn.Level))
	if slug := hn.Slug; len(slug) > 0 {
		v.writeContentStart('s')
		v.b.WriteStrings("\"", slug, "\"")
	}
	v.writeContentStart('i')
	v.acceptInlineSlice(hn.Inlines)
	v.b.WriteByte('}')
}

// VisitHRule writes JSON code for a horizontal rule: <hr>.
func (v *detailVisitor) VisitHRule(hn *ast.HRuleNode) {
	v.writeNodeStart("Hrule")
	v.visitAttributes(hn.Attrs)
	v.b.WriteByte('}')
}

var listCode = map[ast.NestedListCode]string{
	ast.NestedListOrdered:   "OrderedList",
	ast.NestedListUnordered: "BulletList",
	ast.NestedListQuote:     "QuoteList",
}

// VisitNestedList writes JSON code for lists and blockquotes.
func (v *detailVisitor) VisitNestedList(ln *ast.NestedListNode) {
	v.writeNodeStart(listCode[ln.Code])
	v.writeContentStart('c')
	for i, item := range ln.Items {
		if i > 0 {
			v.b.WriteByte(',')
		}
		v.acceptItemSlice(item)
	}
	v.b.WriteString("]}")
}

// VisitDescriptionList emits a JSON description list.
func (v *detailVisitor) VisitDescriptionList(dn *ast.DescriptionListNode) {
	v.writeNodeStart("DescriptionList")
	v.writeContentStart('g')
	for i, def := range dn.Descriptions {
		if i > 0 {
			v.b.WriteByte(',')
		}
		v.b.WriteByte('[')
		v.acceptInlineSlice(def.Term)

		if len(def.Descriptions) > 0 {
			for _, b := range def.Descriptions {
				v.b.WriteByte(',')
				v.acceptDescriptionSlice(b)
			}
		}
		v.b.WriteByte(']')
	}
	v.b.WriteString("]}")
}

// VisitTable emits a JSON table.
func (v *detailVisitor) VisitTable(tn *ast.TableNode) {
	v.writeNodeStart("Table")
	v.writeContentStart('p')

	// Table header
	v.b.WriteByte('[')
	for i, cell := range tn.Header {
		if i > 0 {
			v.b.WriteByte(',')
		}
		v.writeCell(cell)
	}
	v.b.WriteString("],")

	// Table rows
	v.b.WriteByte('[')
	for i, row := range tn.Rows {
		if i > 0 {
			v.b.WriteByte(',')
		}
		v.b.WriteByte('[')
		for j, cell := range row {
			if j > 0 {
				v.b.WriteByte(',')
			}
			v.writeCell(cell)
		}
		v.b.WriteByte(']')
	}
	v.b.WriteString("]]}")
}

var alignmentCode = map[ast.Alignment]string{
	ast.AlignDefault: "[\"\",",
	ast.AlignLeft:    "[\"<\",",
	ast.AlignCenter:  "[\":\",",
	ast.AlignRight:   "[\">\",",
}

func (v *detailVisitor) writeCell(cell *ast.TableCell) {
	v.b.WriteString(alignmentCode[cell.Align])
	v.acceptInlineSlice(cell.Inlines)
	v.b.WriteByte(']')
}

// VisitBLOB writes the binary object as a value.
func (v *detailVisitor) VisitBLOB(bn *ast.BLOBNode) {
	v.writeNodeStart("Blob")
	v.writeContentStart('q')
	writeEscaped(&v.b, bn.Title)
	v.writeContentStart('s')
	writeEscaped(&v.b, bn.Syntax)
	v.writeContentStart('o')
	v.b.WriteBase64(bn.Blob)
	v.b.WriteString("\"}")
}

// VisitText writes text content.
func (v *detailVisitor) VisitText(tn *ast.TextNode) {
	v.writeNodeStart("Text")
	v.writeContentStart('s')
	writeEscaped(&v.b, tn.Text)
	v.b.WriteByte('}')
}

// VisitTag writes tag content.
func (v *detailVisitor) VisitTag(tn *ast.TagNode) {
	v.writeNodeStart("Tag")
	v.writeContentStart('s')
	writeEscaped(&v.b, tn.Tag)
	v.b.WriteByte('}')
}

// VisitSpace emits a white space.
func (v *detailVisitor) VisitSpace(sn *ast.SpaceNode) {
	v.writeNodeStart("Space")
	if l := len(sn.Lexeme); l > 1 {
		v.writeContentStart('n')
		v.b.WriteString(strconv.Itoa(l))
	}
	v.b.WriteByte('}')
}

// VisitBreak writes JSON code for line breaks.
func (v *detailVisitor) VisitBreak(bn *ast.BreakNode) {
	if bn.Hard {
		v.writeNodeStart("Hard")
	} else {
		v.writeNodeStart("Soft")
	}
	v.b.WriteByte('}')
}

var mapRefState = map[ast.RefState]string{
	ast.RefStateInvalid:  "invalid",
	ast.RefStateZettel:   "zettel",
	ast.RefStateSelf:     "self",
	ast.RefStateFound:    "zettel",
	ast.RefStateBroken:   "broken",
	ast.RefStateHosted:   "local",
	ast.RefStateBased:    "based",
	ast.RefStateExternal: "external",
}

// VisitLink writes JSON code for links.
func (v *detailVisitor) VisitLink(ln *ast.LinkNode) {
	if adapt := v.enc.adaptLink; adapt != nil {
		n := adapt(ln)
		var ok bool
		if ln, ok = n.(*ast.LinkNode); !ok {
			n.Accept(v)
			return
		}
	}
	v.writeNodeStart("Link")
	v.visitAttributes(ln.Attrs)
	v.writeContentStart('q')
	writeEscaped(&v.b, mapRefState[ln.Ref.State])
	v.writeContentStart('s')
	writeEscaped(&v.b, ln.Ref.String())
	v.writeContentStart('i')
	v.acceptInlineSlice(ln.Inlines)
	v.b.WriteByte('}')
}

// VisitImage writes JSON code for images.
func (v *detailVisitor) VisitImage(in *ast.ImageNode) {
	if adapt := v.enc.adaptImage; adapt != nil {
		n := adapt(in)
		var ok bool
		if in, ok = n.(*ast.ImageNode); !ok {
			n.Accept(v)
			return
		}
	}
	v.writeNodeStart("Image")
	v.visitAttributes(in.Attrs)
	if in.Ref == nil {
		v.writeContentStart('j')
		v.b.WriteString("\"s\":")
		writeEscaped(&v.b, in.Syntax)
		switch in.Syntax {
		case "svg":
			v.writeContentStart('q')
			writeEscaped(&v.b, string(in.Blob))
		default:
			v.writeContentStart('o')
			v.b.WriteBase64(in.Blob)
			v.b.WriteByte('"')
		}
		v.b.WriteByte('}')
	} else {
		v.writeContentStart('s')
		writeEscaped(&v.b, in.Ref.String())
	}
	if len(in.Inlines) > 0 {
		v.writeContentStart('i')
		v.acceptInlineSlice(in.Inlines)
	}
	v.b.WriteByte('}')
}

// VisitCite writes code for citations.
func (v *detailVisitor) VisitCite(cn *ast.CiteNode) {
	v.writeNodeStart("Cite")
	v.visitAttributes(cn.Attrs)
	v.writeContentStart('s')
	writeEscaped(&v.b, cn.Key)
	if len(cn.Inlines) > 0 {
		v.writeContentStart('i')
		v.acceptInlineSlice(cn.Inlines)
	}
	v.b.WriteByte('}')
}

// VisitFootnote write JSON code for a footnote.
func (v *detailVisitor) VisitFootnote(fn *ast.FootnoteNode) {
	v.writeNodeStart("Footnote")
	v.visitAttributes(fn.Attrs)
	v.writeContentStart('i')
	v.acceptInlineSlice(fn.Inlines)
	v.b.WriteByte('}')
}

// VisitMark writes JSON code to mark a position.
func (v *detailVisitor) VisitMark(mn *ast.MarkNode) {
	v.writeNodeStart("Mark")
	if len(mn.Text) > 0 {
		v.writeContentStart('s')
		writeEscaped(&v.b, mn.Text)
	}
	v.b.WriteByte('}')
}

var formatCode = map[ast.FormatCode]string{
	ast.FormatItalic:    "Italic",
	ast.FormatEmph:      "Emph",
	ast.FormatBold:      "Bold",
	ast.FormatStrong:    "Strong",
	ast.FormatMonospace: "Mono",
	ast.FormatStrike:    "Strikethrough",
	ast.FormatDelete:    "Delete",
	ast.FormatUnder:     "Underline",
	ast.FormatInsert:    "Insert",
	ast.FormatSuper:     "Super",
	ast.FormatSub:       "Sub",
	ast.FormatQuote:     "Quote",
	ast.FormatQuotation: "Quotation",
	ast.FormatSmall:     "Small",
	ast.FormatSpan:      "Span",
}

// VisitFormat write JSON code for formatting text.
func (v *detailVisitor) VisitFormat(fn *ast.FormatNode) {
	v.writeNodeStart(formatCode[fn.Code])
	v.visitAttributes(fn.Attrs)
	v.writeContentStart('i')
	v.acceptInlineSlice(fn.Inlines)
	v.b.WriteByte('}')
}

var literalCode = map[ast.LiteralCode]string{
	ast.LiteralProg:    "Code",
	ast.LiteralKeyb:    "Input",
	ast.LiteralOutput:  "Output",
	ast.LiteralComment: "Comment",
	ast.LiteralHTML:    "HTML",
}

// VisitLiteral write JSON code for literal inline text.
func (v *detailVisitor) VisitLiteral(ln *ast.LiteralNode) {
	code, ok := literalCode[ln.Code]
	if !ok {
		panic(fmt.Sprintf("Unknown literal code %v", ln.Code))
	}
	v.writeNodeStart(code)
	v.visitAttributes(ln.Attrs)
	v.writeContentStart('s')
	writeEscaped(&v.b, ln.Text)
	v.b.WriteByte('}')
}

func (v *detailVisitor) acceptBlockSlice(bns ast.BlockSlice) {
	v.b.WriteByte('[')
	for i, bn := range bns {
		if i > 0 {
			v.b.WriteByte(',')
		}
		bn.Accept(v)
	}
	v.b.WriteByte(']')
}

func (v *detailVisitor) acceptItemSlice(ins ast.ItemSlice) {
	v.b.WriteByte('[')
	for i, in := range ins {
		if i > 0 {
			v.b.WriteByte(',')
		}
		in.Accept(v)
	}
	v.b.WriteByte(']')
}

func (v *detailVisitor) acceptDescriptionSlice(dns ast.DescriptionSlice) {
	v.b.WriteByte('[')
	for i, dn := range dns {
		if i > 0 {
			v.b.WriteByte(',')
		}
		dn.Accept(v)
	}
	v.b.WriteByte(']')
}

func (v *detailVisitor) acceptInlineSlice(ins ast.InlineSlice) {
	v.b.WriteByte('[')
	for i, in := range ins {
		if i > 0 {
			v.b.WriteByte(',')
		}
		in.Accept(v)
	}
	v.b.WriteByte(']')
}

// visitAttributes write JSON attributes
func (v *detailVisitor) visitAttributes(a *ast.Attributes) {
	if a == nil || len(a.Attrs) == 0 {
		return
	}
	keys := make([]string, 0, len(a.Attrs))
	for k := range a.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	v.b.WriteString(",\"a\":{\"")
	for i, k := range keys {
		if i > 0 {
			v.b.WriteString("\",\"")
		}
		v.b.Write(Escape(k))
		v.b.WriteString("\":\"")
		v.b.Write(Escape(a.Attrs[k]))
	}
	v.b.WriteString("\"}")
}

func (v *detailVisitor) writeNodeStart(t string) {
	v.b.WriteStrings("{\"t\":\"", t, "\"")
}

var contentCode = map[rune][]byte{
	'b': []byte(",\"b\":"),   // List of blocks
	'c': []byte(",\"c\":["),  // List of list of blocks
	'g': []byte(",\"g\":["),  // General list
	'i': []byte(",\"i\":"),   // List of inlines
	'j': []byte(",\"j\":{"),  // Embedded JSON object
	'l': []byte(",\"l\":["),  // List of lines
	'n': []byte(",\"n\":"),   // Number
	'o': []byte(",\"o\":\""), // Byte object
	'p': []byte(",\"p\":["),  // Generic tuple
	'q': []byte(",\"q\":"),   // String, if 's' is also needed
	's': []byte(",\"s\":"),   // String
	't': []byte("Content code 't' is not allowed"),
	'y': []byte("Content code 'y' is not allowed"), // field after 'j'
}

func (v *detailVisitor) writeContentStart(code rune) {
	if b, ok := contentCode[code]; ok {
		v.b.Write(b)
		return
	}
	panic("Unknown content code " + strconv.Itoa(int(code)))
}

func (v *detailVisitor) writeMeta(m *meta.Meta, withTitle bool) {
	first := withTitle
	for _, p := range m.Pairs(true) {
		if p.Key == "title" && !withTitle {
			continue
		}
		if first {
			v.b.WriteByte('"')
			first = false
		} else {
			v.b.WriteString(",\"")
		}
		v.b.Write(Escape(p.Key))
		v.b.WriteString("\":")
		if m.Type(p.Key).IsSet {
			v.writeSetValue(p.Value)
		} else {
			v.b.WriteByte('"')
			v.b.Write(Escape(p.Value))
			v.b.WriteByte('"')
		}
	}
}

func (v *detailVisitor) writeSetValue(value string) {
	v.b.WriteByte('[')
	for i, val := range meta.ListFromValue(value) {
		if i > 0 {
			v.b.WriteByte(',')
		}
		v.b.WriteByte('"')
		v.b.Write(Escape(val))
		v.b.WriteByte('"')
	}
	v.b.WriteByte(']')
}
