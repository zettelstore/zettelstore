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
	"zettelstore.de/z/encoder/encfun"
)

func init() {
	encoder.Register("djson", encoder.Info{
		Create: func(env *encoder.Environment) encoder.Encoder { return &jsonDetailEncoder{env: env} },
	})
}

type jsonDetailEncoder struct {
	env *encoder.Environment
}

// WriteZettel writes the encoded zettel to the writer.
func (je *jsonDetailEncoder) WriteZettel(w io.Writer, zn *ast.ZettelNode, inhMeta bool) (int, error) {
	v := newDetailVisitor(w, je)
	v.b.WriteString("{\"meta\":{\"title\":")
	v.walkInlineSlice(encfun.MetaAsInlineSlice(zn.InhMeta, meta.KeyTitle))
	if inhMeta {
		v.writeMeta(zn.InhMeta)
	} else {
		v.writeMeta(zn.Meta)
	}
	v.b.WriteByte('}')
	v.b.WriteString(",\"content\":")
	v.walkBlockSlice(zn.Ast)
	v.b.WriteByte('}')
	length, err := v.b.Flush()
	return length, err
}

// WriteMeta encodes meta data as JSON.
func (je *jsonDetailEncoder) WriteMeta(w io.Writer, m *meta.Meta) (int, error) {
	v := newDetailVisitor(w, je)
	v.b.WriteString("{\"title\":")
	v.walkInlineSlice(encfun.MetaAsInlineSlice(m, meta.KeyTitle))
	v.writeMeta(m)
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
	v.walkBlockSlice(bs)
	length, err := v.b.Flush()
	return length, err
}

// WriteInlines writes an inline slice to the writer
func (je *jsonDetailEncoder) WriteInlines(w io.Writer, is ast.InlineSlice) (int, error) {
	v := newDetailVisitor(w, je)
	v.walkInlineSlice(is)
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
	case *ast.ParaNode:
		v.writeNodeStart("Para")
		v.writeContentStart('i')
		v.walkInlineSlice(n.Inlines)
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
		n, n2 := v.env.AdaptLink(n)
		if n2 != nil {
			ast.Walk(v, n2)
			return nil
		}
		v.writeNodeStart("Link")
		v.visitAttributes(n.Attrs)
		v.writeContentStart('q')
		writeEscaped(&v.b, mapRefState[n.Ref.State])
		v.writeContentStart('s')
		writeEscaped(&v.b, n.Ref.String())
		v.writeContentStart('i')
		v.walkInlineSlice(n.Inlines)
	case *ast.ImageNode:
		v.visitImage(n)
	case *ast.CiteNode:
		v.writeNodeStart("Cite")
		v.visitAttributes(n.Attrs)
		v.writeContentStart('s')
		writeEscaped(&v.b, n.Key)
		if len(n.Inlines) > 0 {
			v.writeContentStart('i')
			v.walkInlineSlice(n.Inlines)
		}
	case *ast.FootnoteNode:
		v.writeNodeStart("Footnote")
		v.visitAttributes(n.Attrs)
		v.writeContentStart('i')
		v.walkInlineSlice(n.Inlines)
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
		v.walkInlineSlice(n.Inlines)
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
		if i > 0 {
			v.b.WriteByte(',')
		}
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
	v.walkBlockSlice(rn.Blocks)
	if len(rn.Inlines) > 0 {
		v.writeContentStart('i')
		v.walkInlineSlice(rn.Inlines)
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
	v.walkInlineSlice(hn.Inlines)
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
		if i > 0 {
			v.b.WriteByte(',')
		}
		v.b.WriteByte('[')
		for j, in := range item {
			if j > 0 {
				v.b.WriteByte(',')
			}
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
		if i > 0 {
			v.b.WriteByte(',')
		}
		v.b.WriteByte('[')
		v.walkInlineSlice(def.Term)

		if len(def.Descriptions) > 0 {
			for _, b := range def.Descriptions {
				v.b.WriteByte(',')
				v.b.WriteByte('[')
				for j, dn := range b {
					if j > 0 {
						v.b.WriteByte(',')
					}
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
	v.walkInlineSlice(cell.Inlines)
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

func (v *detailVisitor) visitImage(in *ast.ImageNode) {
	in, n := v.env.AdaptImage(in)
	if n != nil {
		ast.Walk(v, n)
		return
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
		v.walkInlineSlice(in.Inlines)
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
		if i > 0 {
			v.b.WriteByte(',')
		}
		ast.Walk(v, bn)
	}
	v.b.WriteByte(']')
}

func (v *detailVisitor) walkInlineSlice(ins ast.InlineSlice) {
	v.b.WriteByte('[')
	for i, in := range ins {
		if i > 0 {
			v.b.WriteByte(',')
		}
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

func (v *detailVisitor) writeMeta(m *meta.Meta) {
	for _, p := range m.Pairs(true) {
		if p.Key == meta.KeyTitle {
			continue
		}
		v.b.WriteString(",\"")
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
