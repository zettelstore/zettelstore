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

	"zettelstore.de/z/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/encoder/encfun"
	"zettelstore.de/z/strfun"
)

func init() {
	encoder.Register(api.EncoderDJSON, encoder.Info{
		Create: func(env *encoder.Environment) encoder.Encoder { return &jsonDetailEncoder{env: env} },
	})
}

type jsonDetailEncoder struct {
	env *encoder.Environment
}

// WriteZettel writes the encoded zettel to the writer.
func (je *jsonDetailEncoder) WriteZettel(w io.Writer, zn *ast.ZettelNode) (int, error) {
	v := newDetailVisitor(w, je)
	v.b.WriteString("{\"meta\":{\"title\":")
	ast.Walk(v, encfun.MetaAsInlineList(zn.InhMeta, meta.KeyTitle))
	v.writeMeta(zn.InhMeta)
	v.b.WriteByte('}')
	v.b.WriteString(",\"content\":")
	ast.Walk(v, zn.Ast)
	v.b.WriteByte('}')
	length, err := v.b.Flush()
	return length, err
}

// WriteMeta encodes meta data as JSON.
func (je *jsonDetailEncoder) WriteMeta(w io.Writer, m *meta.Meta) (int, error) {
	v := newDetailVisitor(w, je)
	v.b.WriteString("{\"title\":")
	ast.Walk(v, encfun.MetaAsInlineList(m, meta.KeyTitle))
	v.writeMeta(m)
	v.b.WriteByte('}')
	length, err := v.b.Flush()
	return length, err
}

func (je *jsonDetailEncoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return je.WriteBlocks(w, zn.Ast)
}

// WriteBlocks writes a block slice to the writer
func (je *jsonDetailEncoder) WriteBlocks(w io.Writer, bln *ast.BlockListNode) (int, error) {
	v := newDetailVisitor(w, je)
	ast.Walk(v, bln)
	length, err := v.b.Flush()
	return length, err
}

// WriteInlines writes an inline slice to the writer
func (je *jsonDetailEncoder) WriteInlines(w io.Writer, iln *ast.InlineListNode) (int, error) {
	v := newDetailVisitor(w, je)
	ast.Walk(v, iln)
	length, err := v.b.Flush()
	return length, err
}

// detailVisitor writes the abstract syntax tree to an io.Writer.
type detailVisitor struct {
	b   encoder.BufWriter
	env *encoder.Environment
}

func newDetailVisitor(w io.Writer, je *jsonDetailEncoder) *detailVisitor {
	return &detailVisitor{b: encoder.NewBufWriter(w), env: je.env}
}

func (v *detailVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.BlockListNode:
		v.walkBlockSlice(n.List)
		return nil
	case *ast.InlineListNode:
		v.walkInlineSlice(n.List)
		return nil
	case *ast.ParaNode:
		v.writeNodeStart("Para")
		v.writeContentStart('i')
		ast.Walk(v, n.Inlines)
	case *ast.VerbatimNode:
		v.visitVerbatim(n)
	case *ast.RegionNode:
		v.visitRegion(n)
	case *ast.HeadingNode:
		v.visitHeading(n)
	case *ast.HRuleNode:
		v.writeNodeStart("Hrule")
		v.visitAttributes(n.Attrs)
	case *ast.NestedListNode:
		v.visitNestedList(n)
	case *ast.DescriptionListNode:
		v.visitDescriptionList(n)
	case *ast.TableNode:
		v.visitTable(n)
	case *ast.BLOBNode:
		v.writeNodeStart("Blob")
		v.writeContentStart('q')
		writeEscaped(&v.b, n.Title)
		v.writeContentStart('s')
		writeEscaped(&v.b, n.Syntax)
		v.writeContentStart('o')
		v.b.WriteBase64(n.Blob)
		v.b.WriteByte('"')
	case *ast.TextNode:
		v.writeNodeStart("Text")
		v.writeContentStart('s')
		writeEscaped(&v.b, n.Text)
	case *ast.TagNode:
		v.writeNodeStart("Tag")
		v.writeContentStart('s')
		writeEscaped(&v.b, n.Tag)
	case *ast.SpaceNode:
		v.writeNodeStart("Space")
		if l := len(n.Lexeme); l > 1 {
			v.writeContentStart('n')
			v.b.WriteString(strconv.Itoa(l))
		}
	case *ast.BreakNode:
		if n.Hard {
			v.writeNodeStart("Hard")
		} else {
			v.writeNodeStart("Soft")
		}
	case *ast.LinkNode:
		v.writeNodeStart("Link")
		v.visitAttributes(n.Attrs)
		v.writeContentStart('q')
		writeEscaped(&v.b, mapRefState[n.Ref.State])
		v.writeContentStart('s')
		writeEscaped(&v.b, n.Ref.String())
		v.writeContentStart('i')
		ast.Walk(v, n.Inlines)
	case *ast.EmbedNode:
		v.visitEmbed(n)
	case *ast.CiteNode:
		v.writeNodeStart("Cite")
		v.visitAttributes(n.Attrs)
		v.writeContentStart('s')
		writeEscaped(&v.b, n.Key)
		if n.Inlines != nil {
			v.writeContentStart('i')
			ast.Walk(v, n.Inlines)
		}
	case *ast.FootnoteNode:
		v.writeNodeStart("Footnote")
		v.visitAttributes(n.Attrs)
		v.writeContentStart('i')
		ast.Walk(v, n.Inlines)
	case *ast.MarkNode:
		v.writeNodeStart("Mark")
		if len(n.Text) > 0 {
			v.writeContentStart('s')
			writeEscaped(&v.b, n.Text)
		}
	case *ast.FormatNode:
		v.writeNodeStart(mapFormatKind[n.Kind])
		v.visitAttributes(n.Attrs)
		v.writeContentStart('i')
		ast.Walk(v, n.Inlines)
	case *ast.LiteralNode:
		kind, ok := mapLiteralKind[n.Kind]
		if !ok {
			panic(fmt.Sprintf("Unknown literal kind %v", n.Kind))
		}
		v.writeNodeStart(kind)
		v.visitAttributes(n.Attrs)
		v.writeContentStart('s')
		writeEscaped(&v.b, n.Text)
	default:
		return v
	}
	v.b.WriteByte('}')
	return nil
}

var mapVerbatimKind = map[ast.VerbatimKind]string{
	ast.VerbatimProg:    "CodeBlock",
	ast.VerbatimComment: "CommentBlock",
	ast.VerbatimHTML:    "HTMLBlock",
}

func (v *detailVisitor) visitVerbatim(vn *ast.VerbatimNode) {
	kind, ok := mapVerbatimKind[vn.Kind]
	if !ok {
		panic(fmt.Sprintf("Unknown verbatim kind %v", vn.Kind))
	}
	v.writeNodeStart(kind)
	v.visitAttributes(vn.Attrs)
	v.writeContentStart('l')
	for i, line := range vn.Lines {
		v.writeComma(i)
		writeEscaped(&v.b, line)
	}
	v.b.WriteByte(']')
}

var mapRegionKind = map[ast.RegionKind]string{
	ast.RegionSpan:  "SpanBlock",
	ast.RegionQuote: "QuoteBlock",
	ast.RegionVerse: "VerseBlock",
}

func (v *detailVisitor) visitRegion(rn *ast.RegionNode) {
	kind, ok := mapRegionKind[rn.Kind]
	if !ok {
		panic(fmt.Sprintf("Unknown region kind %v", rn.Kind))
	}
	v.writeNodeStart(kind)
	v.visitAttributes(rn.Attrs)
	v.writeContentStart('b')
	ast.Walk(v, rn.Blocks)
	if rn.Inlines != nil {
		v.writeContentStart('i')
		ast.Walk(v, rn.Inlines)
	}
}

func (v *detailVisitor) visitHeading(hn *ast.HeadingNode) {
	v.writeNodeStart("Heading")
	v.visitAttributes(hn.Attrs)
	v.writeContentStart('n')
	v.b.WriteString(strconv.Itoa(hn.Level))
	if slug := hn.Slug; len(slug) > 0 {
		v.writeContentStart('s')
		v.b.WriteStrings("\"", slug, "\"")
	}
	v.writeContentStart('i')
	ast.Walk(v, hn.Inlines)
}

var mapNestedListKind = map[ast.NestedListKind]string{
	ast.NestedListOrdered:   "OrderedList",
	ast.NestedListUnordered: "BulletList",
	ast.NestedListQuote:     "QuoteList",
}

func (v *detailVisitor) visitNestedList(ln *ast.NestedListNode) {
	v.writeNodeStart(mapNestedListKind[ln.Kind])
	v.writeContentStart('c')
	for i, item := range ln.Items {
		v.writeComma(i)
		v.b.WriteByte('[')
		for j, in := range item {
			v.writeComma(j)
			ast.Walk(v, in)
		}
		v.b.WriteByte(']')
	}
	v.b.WriteByte(']')
}

func (v *detailVisitor) visitDescriptionList(dn *ast.DescriptionListNode) {
	v.writeNodeStart("DescriptionList")
	v.writeContentStart('g')
	for i, def := range dn.Descriptions {
		v.writeComma(i)
		v.b.WriteByte('[')
		ast.Walk(v, def.Term)

		if len(def.Descriptions) > 0 {
			for _, b := range def.Descriptions {
				v.b.WriteString(",[")
				for j, dn := range b {
					v.writeComma(j)
					ast.Walk(v, dn)
				}
				v.b.WriteByte(']')
			}
		}
		v.b.WriteByte(']')
	}
	v.b.WriteByte(']')
}

func (v *detailVisitor) visitTable(tn *ast.TableNode) {
	v.writeNodeStart("Table")
	v.writeContentStart('p')

	// Table header
	v.b.WriteByte('[')
	for i, cell := range tn.Header {
		v.writeComma(i)
		v.writeCell(cell)
	}
	v.b.WriteString("],")

	// Table rows
	v.b.WriteByte('[')
	for i, row := range tn.Rows {
		v.writeComma(i)
		v.b.WriteByte('[')
		for j, cell := range row {
			v.writeComma(j)
			v.writeCell(cell)
		}
		v.b.WriteByte(']')
	}
	v.b.WriteString("]]")
}

var alignmentCode = map[ast.Alignment]string{
	ast.AlignDefault: "[\"\",",
	ast.AlignLeft:    "[\"<\",",
	ast.AlignCenter:  "[\":\",",
	ast.AlignRight:   "[\">\",",
}

func (v *detailVisitor) writeCell(cell *ast.TableCell) {
	v.b.WriteString(alignmentCode[cell.Align])
	ast.Walk(v, cell.Inlines)
	v.b.WriteByte(']')
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

func (v *detailVisitor) visitEmbed(en *ast.EmbedNode) {
	v.writeNodeStart("Embed")
	v.visitAttributes(en.Attrs)
	switch m := en.Material.(type) {
	case *ast.ReferenceMaterialNode:
		v.writeContentStart('s')
		writeEscaped(&v.b, m.Ref.String())
	case *ast.BLOBMaterialNode:
		v.writeContentStart('j')
		v.b.WriteString("\"s\":")
		writeEscaped(&v.b, m.Syntax)
		switch m.Syntax {
		case "svg":
			v.writeContentStart('q')
			writeEscaped(&v.b, string(m.Blob))
		default:
			v.writeContentStart('o')
			v.b.WriteBase64(m.Blob)
			v.b.WriteByte('"')
		}
		v.b.WriteByte('}')
	default:
		panic(fmt.Sprintf("Unknown material type %t for %v", en.Material, en.Material))
	}

	if en.Inlines != nil {
		v.writeContentStart('i')
		ast.Walk(v, en.Inlines)
	}
}

var mapFormatKind = map[ast.FormatKind]string{
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

var mapLiteralKind = map[ast.LiteralKind]string{
	ast.LiteralProg:    "Code",
	ast.LiteralKeyb:    "Input",
	ast.LiteralOutput:  "Output",
	ast.LiteralComment: "Comment",
	ast.LiteralHTML:    "HTML",
}

func (v *detailVisitor) walkBlockSlice(bns ast.BlockSlice) {
	v.b.WriteByte('[')
	for i, bn := range bns {
		v.writeComma(i)
		ast.Walk(v, bn)
	}
	v.b.WriteByte(']')
}

func (v *detailVisitor) walkInlineSlice(ins ast.InlineSlice) {
	v.b.WriteByte('[')
	for i, in := range ins {
		v.writeComma(i)
		ast.Walk(v, in)
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
		strfun.JSONEscape(&v.b, k)
		v.b.WriteString("\":\"")
		strfun.JSONEscape(&v.b, a.Attrs[k])
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

func (v *detailVisitor) writeMeta(m *meta.Meta) {
	for _, p := range m.Pairs(true) {
		if p.Key == meta.KeyTitle {
			continue
		}
		v.b.WriteString(",\"")
		strfun.JSONEscape(&v.b, p.Key)
		v.b.WriteString("\":")
		if m.Type(p.Key).IsSet {
			v.writeSetValue(p.Value)
		} else {
			writeEscaped(&v.b, p.Value)
		}
	}
}

func (v *detailVisitor) writeSetValue(value string) {
	v.b.WriteByte('[')
	for i, val := range meta.ListFromValue(value) {
		v.writeComma(i)
		writeEscaped(&v.b, val)
	}
	v.b.WriteByte(']')
}

func (v *detailVisitor) writeComma(pos int) {
	if pos > 0 {
		v.b.WriteByte(',')
	}
}

func writeEscaped(b *encoder.BufWriter, s string) {
	b.WriteByte('"')
	strfun.JSONEscape(b, s)
	b.WriteByte('"')
}
