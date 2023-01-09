//-----------------------------------------------------------------------------
// Copyright (c) 2020-2023 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package zjsonenc encodes the abstract syntax tree into JSON.
//
// Deprecated in v0.11
package zjsonenc

import (
	"fmt"
	"io"
	"strconv"

	"zettelstore.de/c/api"
	"zettelstore.de/c/attrs"
	"zettelstore.de/c/zjson"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/strfun"
)

func init() {
	encoder.Register(api.EncoderZJSON, func() encoder.Encoder { return Create() })
}

// Create a ZJSON encoder
func Create() *Encoder { return &myJE }

type Encoder struct{}

var myJE Encoder

// WriteZettel writes the encoded zettel to the writer.
func (*Encoder) WriteZettel(w io.Writer, zn *ast.ZettelNode, evalMeta encoder.EvalMetaFunc) (int, error) {
	v := newDetailVisitor(w)
	v.b.WriteString(`{"meta":`)
	v.writeMeta(zn.InhMeta, evalMeta)
	v.b.WriteString(`,"content":`)
	ast.Walk(v, &zn.Ast)
	v.b.WriteByte('}')
	length, err := v.b.Flush()
	return length, err
}

// WriteMeta encodes meta data as JSON.
func (*Encoder) WriteMeta(w io.Writer, m *meta.Meta, evalMeta encoder.EvalMetaFunc) (int, error) {
	v := newDetailVisitor(w)
	v.writeMeta(m, evalMeta)
	length, err := v.b.Flush()
	return length, err
}

func (je *Encoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return je.WriteBlocks(w, &zn.Ast)
}

// WriteBlocks writes a block slice to the writer
func (*Encoder) WriteBlocks(w io.Writer, bs *ast.BlockSlice) (int, error) {
	v := newDetailVisitor(w)
	ast.Walk(v, bs)
	length, err := v.b.Flush()
	return length, err
}

// WriteInlines writes an inline slice to the writer
func (*Encoder) WriteInlines(w io.Writer, is *ast.InlineSlice) (int, error) {
	v := newDetailVisitor(w)
	ast.Walk(v, is)
	length, err := v.b.Flush()
	return length, err
}

// visitor writes the abstract syntax tree to an io.Writer.
type visitor struct {
	b       encoder.EncWriter
	inVerse bool // Visiting a verse block: save spaces in ZJSON object
}

func newDetailVisitor(w io.Writer) *visitor { return &visitor{b: encoder.NewEncWriter(w)} }

func (v *visitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.BlockSlice:
		v.visitBlockSlice(n)
		return nil
	case *ast.InlineSlice:
		v.walkInlineSlice(n)
		return nil
	case *ast.ParaNode:
		v.writeNodeStart(zjson.TypeParagraph)
		v.writeContentStart(zjson.NameInline)
		ast.Walk(v, &n.Inlines)
	case *ast.VerbatimNode:
		v.visitVerbatim(n)
	case *ast.RegionNode:
		v.visitRegion(n)
	case *ast.HeadingNode:
		v.visitHeading(n)
	case *ast.HRuleNode:
		v.writeNodeStart(zjson.TypeBreakThematic)
		v.visitAttributes(n.Attrs)
	case *ast.NestedListNode:
		v.visitNestedList(n)
	case *ast.DescriptionListNode:
		v.visitDescriptionList(n)
	case *ast.TableNode:
		v.visitTable(n)
	case *ast.TranscludeNode:
		v.writeNodeStart(zjson.TypeTransclude)
		v.visitAttributes(n.Attrs)
		v.writeContentStart(zjson.NameString2)
		writeEscaped(&v.b, mapRefState[n.Ref.State])
		v.writeContentStart(zjson.NameString)
		writeEscaped(&v.b, n.Ref.String())
	case *ast.BLOBNode:
		v.visitBLOB(n)
	case *ast.TextNode:
		v.writeNodeStart(zjson.TypeText)
		v.writeContentStart(zjson.NameString)
		writeEscaped(&v.b, n.Text)
	case *ast.SpaceNode:
		v.writeNodeStart(zjson.TypeSpace)
		if v.inVerse {
			v.writeContentStart(zjson.NameString)
			writeEscaped(&v.b, n.Lexeme)
		}
	case *ast.BreakNode:
		if n.Hard {
			v.writeNodeStart(zjson.TypeBreakHard)
		} else {
			v.writeNodeStart(zjson.TypeBreakSoft)
		}
	case *ast.LinkNode:
		v.visitLink(n)
	case *ast.EmbedRefNode:
		v.visitEmbedRef(n)
	case *ast.EmbedBLOBNode:
		v.visitEmbedBLOB(n)
	case *ast.CiteNode:
		v.writeNodeStart(zjson.TypeCitation)
		v.visitAttributes(n.Attrs)
		v.writeContentStart(zjson.NameString)
		writeEscaped(&v.b, n.Key)
		if len(n.Inlines) > 0 {
			v.writeContentStart(zjson.NameInline)
			ast.Walk(v, &n.Inlines)
		}
	case *ast.FootnoteNode:
		v.writeNodeStart(zjson.TypeFootnote)
		v.visitAttributes(n.Attrs)
		v.writeContentStart(zjson.NameInline)
		ast.Walk(v, &n.Inlines)
	case *ast.MarkNode:
		v.visitMark(n)
	case *ast.FormatNode:
		v.writeNodeStart(mapFormatKind[n.Kind])
		v.visitAttributes(n.Attrs)
		v.writeContentStart(zjson.NameInline)
		ast.Walk(v, &n.Inlines)
	case *ast.LiteralNode:
		kind, ok := mapLiteralKind[n.Kind]
		if !ok {
			panic(fmt.Sprintf("Unknown literal kind %v", n.Kind))
		}
		v.writeNodeStart(kind)
		v.visitAttributes(n.Attrs)
		v.writeContentStart(zjson.NameString)
		writeEscaped(&v.b, string(n.Content))
	default:
		return v
	}
	v.b.WriteByte('}')
	return nil
}

var mapVerbatimKind = map[ast.VerbatimKind]string{
	ast.VerbatimZettel:  zjson.TypeVerbatimZettel,
	ast.VerbatimProg:    zjson.TypeVerbatimCode,
	ast.VerbatimEval:    zjson.TypeVerbatimEval,
	ast.VerbatimMath:    zjson.TypeVerbatimMath,
	ast.VerbatimComment: zjson.TypeVerbatimComment,
	ast.VerbatimHTML:    zjson.TypeVerbatimHTML,
}

func (v *visitor) visitVerbatim(vn *ast.VerbatimNode) {
	kind, ok := mapVerbatimKind[vn.Kind]
	if !ok {
		panic(fmt.Sprintf("Unknown verbatim kind %v", vn.Kind))
	}
	v.writeNodeStart(kind)
	v.visitAttributes(vn.Attrs)
	v.writeContentStart(zjson.NameString)
	writeEscaped(&v.b, string(vn.Content))
}

var mapRegionKind = map[ast.RegionKind]string{
	ast.RegionSpan:  zjson.TypeBlock,
	ast.RegionQuote: zjson.TypeExcerpt,
	ast.RegionVerse: zjson.TypePoem,
}

func (v *visitor) visitRegion(rn *ast.RegionNode) {
	kind, ok := mapRegionKind[rn.Kind]
	if !ok {
		panic(fmt.Sprintf("Unknown region kind %v", rn.Kind))
	}
	saveInVerse := v.inVerse
	if rn.Kind == ast.RegionVerse {
		v.inVerse = true
	}
	v.writeNodeStart(kind)
	v.visitAttributes(rn.Attrs)
	v.writeContentStart(zjson.NameBlock)
	ast.Walk(v, &rn.Blocks)
	if len(rn.Inlines) > 0 {
		v.writeContentStart(zjson.NameInline)
		ast.Walk(v, &rn.Inlines)
	}
	v.inVerse = saveInVerse
}

func (v *visitor) visitHeading(hn *ast.HeadingNode) {
	v.writeNodeStart(zjson.TypeHeading)
	v.visitAttributes(hn.Attrs)
	v.writeContentStart(zjson.NameNumeric)
	v.b.WriteString(strconv.Itoa(hn.Level))
	if fragment := hn.Fragment; fragment != "" {
		v.writeContentStart(zjson.NameString)
		v.b.WriteStrings(`"`, fragment, `"`)
	}
	v.writeContentStart(zjson.NameInline)
	ast.Walk(v, &hn.Inlines)
}

var mapNestedListKind = map[ast.NestedListKind]string{
	ast.NestedListOrdered:   zjson.TypeListOrdered,
	ast.NestedListUnordered: zjson.TypeListBullet,
	ast.NestedListQuote:     zjson.TypeListQuotation,
}

func (v *visitor) visitNestedList(ln *ast.NestedListNode) {
	v.writeNodeStart(mapNestedListKind[ln.Kind])
	v.writeContentStart(zjson.NameList)
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

func (v *visitor) visitDescriptionList(dn *ast.DescriptionListNode) {
	v.writeNodeStart(zjson.TypeDescrList)
	v.writeContentStart(zjson.NameDescrList)
	for i, def := range dn.Descriptions {
		v.writeComma(i)
		v.b.WriteStrings(`{"`, zjson.NameInline, `":`)
		ast.Walk(v, &def.Term)

		if len(def.Descriptions) > 0 {
			v.writeContentStart(zjson.NameDescription)
			for j, b := range def.Descriptions {
				v.writeComma(j)
				v.b.WriteByte('[')
				for k, dn := range b {
					v.writeComma(k)
					ast.Walk(v, dn)
				}
				v.b.WriteByte(']')
			}
			v.b.WriteByte(']')
		}
		v.b.WriteByte('}')
	}
	v.b.WriteByte(']')
}

func (v *visitor) visitTable(tn *ast.TableNode) {
	v.writeNodeStart(zjson.TypeTable)
	v.writeContentStart(zjson.NameTable)

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
	ast.AlignDefault: "",
	ast.AlignLeft:    "<",
	ast.AlignCenter:  ":",
	ast.AlignRight:   ">",
}

func (v *visitor) writeCell(cell *ast.TableCell) {
	if aCode := alignmentCode[cell.Align]; aCode != "" {
		v.b.WriteStrings(`{"`, zjson.NameString, `":"`, aCode, `","`, zjson.NameInline, `":`)
	} else {
		v.b.WriteStrings(`{"`, zjson.NameInline, `":`)
	}
	ast.Walk(v, &cell.Inlines)
	v.b.WriteByte('}')
}

func (v *visitor) visitBLOB(bn *ast.BLOBNode) {
	v.writeNodeStart(zjson.TypeBLOB)
	if bn.Title != "" {
		v.writeContentStart(zjson.NameString2)
		writeEscaped(&v.b, bn.Title)
	}
	v.writeContentStart(zjson.NameString)
	writeEscaped(&v.b, bn.Syntax)
	if bn.Syntax == meta.SyntaxSVG {
		v.writeContentStart(zjson.NameString3)
		writeEscaped(&v.b, string(bn.Blob))
	} else {
		v.writeContentStart(zjson.NameBinary)
		v.b.WriteBase64(bn.Blob)
		v.b.WriteByte('"')
	}
}

var mapRefState = map[ast.RefState]string{
	ast.RefStateInvalid:  zjson.RefStateInvalid,
	ast.RefStateZettel:   zjson.RefStateZettel,
	ast.RefStateSelf:     zjson.RefStateSelf,
	ast.RefStateFound:    zjson.RefStateFound,
	ast.RefStateBroken:   zjson.RefStateBroken,
	ast.RefStateHosted:   zjson.RefStateHosted,
	ast.RefStateBased:    zjson.RefStateBased,
	ast.RefStateQuery:    zjson.RefStateQuery,
	ast.RefStateExternal: zjson.RefStateExternal,
}

func (v *visitor) visitLink(ln *ast.LinkNode) {
	v.writeNodeStart(zjson.TypeLink)
	v.visitAttributes(ln.Attrs)
	v.writeContentStart(zjson.NameString2)
	writeEscaped(&v.b, mapRefState[ln.Ref.State])
	v.writeContentStart(zjson.NameString)
	if ln.Ref.State == ast.RefStateQuery {
		writeEscaped(&v.b, ln.Ref.Value)
	} else {
		writeEscaped(&v.b, ln.Ref.String())
	}
	if len(ln.Inlines) > 0 {
		v.writeContentStart(zjson.NameInline)
		ast.Walk(v, &ln.Inlines)
	}
}

func (v *visitor) visitEmbedRef(en *ast.EmbedRefNode) {
	v.writeNodeStart(zjson.TypeEmbed)
	v.visitAttributes(en.Attrs)
	v.writeContentStart(zjson.NameString)
	writeEscaped(&v.b, en.Ref.String())

	if len(en.Inlines) > 0 {
		v.writeContentStart(zjson.NameInline)
		ast.Walk(v, &en.Inlines)
	}
	if en.Syntax != "" {
		v.writeContentStart(zjson.NameString2)
		writeEscaped(&v.b, en.Syntax)
	}
}

func (v *visitor) visitEmbedBLOB(en *ast.EmbedBLOBNode) {
	v.writeNodeStart(zjson.TypeEmbedBLOB)
	v.visitAttributes(en.Attrs)
	v.writeContentStart(zjson.NameString)
	writeEscaped(&v.b, en.Syntax)
	if en.Syntax == meta.SyntaxSVG {
		v.writeContentStart(zjson.NameString3)
		writeEscaped(&v.b, string(en.Blob))
	} else {
		v.writeContentStart(zjson.NameBinary)
		v.b.WriteBase64(en.Blob)
		v.b.WriteByte('"')
	}
	if len(en.Inlines) > 0 {
		v.writeContentStart(zjson.NameInline)
		ast.Walk(v, &en.Inlines)
	}
}

func (v *visitor) visitMark(mn *ast.MarkNode) {
	v.writeNodeStart(zjson.TypeMark)
	if text := mn.Mark; text != "" {
		v.writeContentStart(zjson.NameString)
		writeEscaped(&v.b, text)
	}
	if fragment := mn.Fragment; fragment != "" {
		v.writeContentStart(zjson.NameString2)
		v.b.WriteByte('"')
		v.b.WriteString(fragment)
		v.b.WriteByte('"')
	}
	if len(mn.Inlines) > 0 {
		v.writeContentStart(zjson.NameInline)
		ast.Walk(v, &mn.Inlines)
	}
}

var mapFormatKind = map[ast.FormatKind]string{
	ast.FormatEmph:   zjson.TypeFormatEmph,
	ast.FormatStrong: zjson.TypeFormatStrong,
	ast.FormatDelete: zjson.TypeFormatDelete,
	ast.FormatInsert: zjson.TypeFormatInsert,
	ast.FormatSuper:  zjson.TypeFormatSuper,
	ast.FormatSub:    zjson.TypeFormatSub,
	ast.FormatQuote:  zjson.TypeFormatQuote,
	ast.FormatSpan:   zjson.TypeFormatSpan,
}

var mapLiteralKind = map[ast.LiteralKind]string{
	ast.LiteralZettel:  zjson.TypeLiteralZettel,
	ast.LiteralProg:    zjson.TypeLiteralCode,
	ast.LiteralInput:   zjson.TypeLiteralInput,
	ast.LiteralOutput:  zjson.TypeLiteralOutput,
	ast.LiteralComment: zjson.TypeLiteralComment,
	ast.LiteralHTML:    zjson.TypeLiteralHTML,
	ast.LiteralMath:    zjson.TypeLiteralMath,
}

func (v *visitor) visitBlockSlice(bs *ast.BlockSlice) {
	v.b.WriteByte('[')
	for i, bn := range *bs {
		v.writeComma(i)
		ast.Walk(v, bn)
	}
	v.b.WriteByte(']')
}

func (v *visitor) walkInlineSlice(is *ast.InlineSlice) {
	v.b.WriteByte('[')
	for i, in := range *is {
		v.writeComma(i)
		ast.Walk(v, in)
	}
	v.b.WriteByte(']')
}

// visitAttributes write JSON attributes
func (v *visitor) visitAttributes(a attrs.Attributes) {
	if a.IsEmpty() {
		return
	}

	v.writeContentStart(zjson.NameAttribute)
	for i, k := range a.Keys() {
		if i > 0 {
			v.b.WriteString(`","`)
		}
		strfun.JSONEscape(&v.b, k)
		v.b.WriteString(`":"`)
		strfun.JSONEscape(&v.b, a[k])
	}
	v.b.WriteString(`"}`)
}

func (v *visitor) writeNodeStart(t string) {
	v.b.WriteStrings(`{"":"`, t, `"`)
}

var valueStart = map[string]string{
	zjson.NameBlock:       "",
	zjson.NameAttribute:   `{"`,
	zjson.NameList:        "[",
	zjson.NameDescrList:   "[",
	zjson.NameDescription: "[",
	zjson.NameInline:      "",
	zjson.NameBLOB:        "{",
	zjson.NameNumeric:     "",
	zjson.NameBinary:      `"`,
	zjson.NameTable:       "[",
	zjson.NameString2:     "",
	zjson.NameString:      "",
	zjson.NameString3:     "",
}

func (v *visitor) writeContentStart(jsonName string) {
	s, ok := valueStart[jsonName]
	if !ok {
		panic("Unknown object name " + jsonName)
	}
	v.b.WriteStrings(`,"`, jsonName, `":`, s)
}

func (v *visitor) writeMeta(m *meta.Meta, evalMeta encoder.EvalMetaFunc) {
	v.b.WriteByte('{')
	for i, p := range m.ComputedPairs() {
		if i > 0 {
			v.b.WriteByte(',')
		}
		v.b.WriteByte('"')
		key := p.Key
		strfun.JSONEscape(&v.b, key)
		t := m.Type(key)
		v.b.WriteStrings(`":{"`, zjson.NameType, `":"`, t.Name, `","`)
		if t.IsSet {
			v.b.WriteStrings(zjson.NameSet, `":`)
			v.writeSetValue(p.Value)
		} else if t == meta.TypeZettelmarkup {
			v.b.WriteStrings(zjson.NameInline, `":`)
			is := evalMeta(p.Value)
			ast.Walk(v, &is)
		} else {
			v.b.WriteStrings(zjson.NameString, `":`)
			writeEscaped(&v.b, p.Value)
		}
		v.b.WriteByte('}')
	}
	v.b.WriteByte('}')
}

func (v *visitor) writeSetValue(value string) {
	v.b.WriteByte('[')
	for i, val := range meta.ListFromValue(value) {
		v.writeComma(i)
		writeEscaped(&v.b, val)
	}
	v.b.WriteByte(']')
}

func (v *visitor) writeComma(pos int) {
	if pos > 0 {
		v.b.WriteByte(',')
	}
}

func writeEscaped(b *encoder.EncWriter, s string) {
	b.WriteByte('"')
	strfun.JSONEscape(b, s)
	b.WriteByte('"')
}
