//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package nativeenc encodes the abstract syntax tree into native format.
package nativeenc

import (
	"fmt"
	"io"
	"sort"
	"strconv"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/encoder/encfun"
	"zettelstore.de/z/parser"
)

func init() {
	encoder.Register("native", encoder.Info{
		Create: func(env *encoder.Environment) encoder.Encoder { return &nativeEncoder{env: env} },
	})
}

type nativeEncoder struct {
	env *encoder.Environment
}

// WriteZettel encodes the zettel to the writer.
func (ne *nativeEncoder) WriteZettel(
	w io.Writer, zn *ast.ZettelNode, inhMeta bool) (int, error) {
	v := newVisitor(w, ne)
	v.b.WriteString("[Title ")
	v.acceptInlineSlice(encfun.MetaAsInlineSlice(zn.InhMeta, meta.KeyTitle))
	v.b.WriteByte(']')
	if inhMeta {
		v.acceptMeta(zn.InhMeta, false)
	} else {
		v.acceptMeta(zn.Meta, false)
	}
	v.b.WriteByte('\n')
	v.acceptBlockSlice(zn.Ast)
	length, err := v.b.Flush()
	return length, err
}

// WriteMeta encodes meta data in native format.
func (ne *nativeEncoder) WriteMeta(w io.Writer, m *meta.Meta) (int, error) {
	v := newVisitor(w, ne)
	v.acceptMeta(m, true)
	length, err := v.b.Flush()
	return length, err
}

func (ne *nativeEncoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return ne.WriteBlocks(w, zn.Ast)
}

// WriteBlocks writes a block slice to the writer
func (ne *nativeEncoder) WriteBlocks(w io.Writer, bs ast.BlockSlice) (int, error) {
	v := newVisitor(w, ne)
	v.acceptBlockSlice(bs)
	length, err := v.b.Flush()
	return length, err
}

// WriteInlines writes an inline slice to the writer
func (ne *nativeEncoder) WriteInlines(w io.Writer, is ast.InlineSlice) (int, error) {
	v := newVisitor(w, ne)
	v.acceptInlineSlice(is)
	length, err := v.b.Flush()
	return length, err
}

// visitor writes the abstract syntax tree to an io.Writer.
type visitor struct {
	b     encoder.BufWriter
	level int
	env   *encoder.Environment
}

func newVisitor(w io.Writer, enc *nativeEncoder) *visitor {
	return &visitor{b: encoder.NewBufWriter(w), env: enc.env}
}

var (
	rawBackslash   = []byte{'\\', '\\'}
	rawDoubleQuote = []byte{'\\', '"'}
	rawNewline     = []byte{'\\', 'n'}
)

func (v *visitor) acceptMeta(m *meta.Meta, withTitle bool) {
	if withTitle {
		v.b.WriteString("[Title ")
		v.acceptInlineSlice(parser.ParseMetadata(m.GetDefault(meta.KeyTitle, "")))
		v.b.WriteByte(']')
	}
	v.writeMetaString(m, meta.KeyRole, "Role")
	v.writeMetaList(m, meta.KeyTags, "Tags")
	v.writeMetaString(m, meta.KeySyntax, "Syntax")
	pairs := m.PairsRest(true)
	if len(pairs) == 0 {
		return
	}
	v.b.WriteString("\n[Header")
	v.level++
	for i, p := range pairs {
		if i > 0 {
			v.b.WriteByte(',')
		}
		v.writeNewLine()
		v.b.WriteByte('[')
		v.b.WriteStrings(p.Key, " \"")
		v.writeEscaped(p.Value)
		v.b.WriteString("\"]")
	}
	v.level--
	v.b.WriteByte(']')
}

func (v *visitor) writeMetaString(m *meta.Meta, key, native string) {
	if val, ok := m.Get(key); ok && len(val) > 0 {
		v.b.WriteStrings("\n[", native, " \"", val, "\"]")
	}
}

func (v *visitor) writeMetaList(m *meta.Meta, key, native string) {
	if vals, ok := m.GetList(key); ok && len(vals) > 0 {
		v.b.WriteStrings("\n[", native)
		for _, val := range vals {
			v.b.WriteByte(' ')
			v.b.WriteString(val)
		}
		v.b.WriteByte(']')
	}
}

// VisitPara emits native code for a paragraph.
func (v *visitor) VisitPara(pn *ast.ParaNode) {
	v.b.WriteString("[Para ")
	v.acceptInlineSlice(pn.Inlines)
	v.b.WriteByte(']')
}

var verbatimCode = map[ast.VerbatimCode][]byte{
	ast.VerbatimProg:    []byte("[CodeBlock"),
	ast.VerbatimComment: []byte("[CommentBlock"),
	ast.VerbatimHTML:    []byte("[HTMLBlock"),
}

// VisitVerbatim emits native code for verbatim lines.
func (v *visitor) VisitVerbatim(vn *ast.VerbatimNode) {
	code, ok := verbatimCode[vn.Code]
	if !ok {
		panic(fmt.Sprintf("Unknown verbatim code %v", vn.Code))
	}
	v.b.Write(code)
	v.visitAttributes(vn.Attrs)
	v.b.WriteString(" \"")
	for i, line := range vn.Lines {
		if i > 0 {
			v.b.Write(rawNewline)
		}
		v.writeEscaped(line)
	}
	v.b.WriteString("\"]")
}

var regionCode = map[ast.RegionCode][]byte{
	ast.RegionSpan:  []byte("[SpanBlock"),
	ast.RegionQuote: []byte("[QuoteBlock"),
	ast.RegionVerse: []byte("[VerseBlock"),
}

// VisitRegion writes native code for block regions.
func (v *visitor) VisitRegion(rn *ast.RegionNode) {
	code, ok := regionCode[rn.Code]
	if !ok {
		panic(fmt.Sprintf("Unknown region code %v", rn.Code))
	}
	v.b.Write(code)
	v.visitAttributes(rn.Attrs)
	v.level++
	v.writeNewLine()
	v.b.WriteByte('[')
	v.level++
	v.acceptBlockSlice(rn.Blocks)
	v.level--
	v.b.WriteByte(']')
	if len(rn.Inlines) > 0 {
		v.b.WriteByte(',')
		v.writeNewLine()
		v.b.WriteString("[Cite ")
		v.acceptInlineSlice(rn.Inlines)
		v.b.WriteByte(']')
	}
	v.level--
	v.b.WriteByte(']')
}

// VisitHeading writes the native code for a heading.
func (v *visitor) VisitHeading(hn *ast.HeadingNode) {
	v.b.WriteStrings("[Heading ", strconv.Itoa(hn.Level), " \"", hn.Slug, "\"")
	v.visitAttributes(hn.Attrs)
	v.b.WriteByte(' ')
	v.acceptInlineSlice(hn.Inlines)
	v.b.WriteByte(']')
}

// VisitHRule writes native code for a horizontal rule: <hr>.
func (v *visitor) VisitHRule(hn *ast.HRuleNode) {
	v.b.WriteString("[Hrule")
	v.visitAttributes(hn.Attrs)
	v.b.WriteByte(']')
}

var listCode = map[ast.NestedListCode][]byte{
	ast.NestedListOrdered:   []byte("[OrderedList"),
	ast.NestedListUnordered: []byte("[BulletList"),
	ast.NestedListQuote:     []byte("[QuoteList"),
}

// VisitNestedList writes native code for lists and blockquotes.
func (v *visitor) VisitNestedList(ln *ast.NestedListNode) {
	v.b.Write(listCode[ln.Code])
	v.level++
	for i, item := range ln.Items {
		if i > 0 {
			v.b.WriteByte(',')
		}
		v.writeNewLine()
		v.level++
		v.b.WriteByte('[')
		v.acceptItemSlice(item)
		v.b.WriteByte(']')
		v.level--
	}
	v.level--
	v.b.WriteByte(']')
}

// VisitDescriptionList emits a native description list.
func (v *visitor) VisitDescriptionList(dn *ast.DescriptionListNode) {
	v.b.WriteString("[DescriptionList")
	v.level++
	for i, descr := range dn.Descriptions {
		if i > 0 {
			v.b.WriteByte(',')
		}
		v.writeNewLine()
		v.b.WriteString("[Term [")
		v.acceptInlineSlice(descr.Term)
		v.b.WriteByte(']')

		if len(descr.Descriptions) > 0 {
			v.level++
			for _, b := range descr.Descriptions {
				v.b.WriteByte(',')
				v.writeNewLine()
				v.b.WriteString("[Description")
				v.level++
				v.writeNewLine()
				v.acceptDescriptionSlice(b)
				v.b.WriteByte(']')
				v.level--
			}
			v.level--
		}
		v.b.WriteByte(']')
	}
	v.level--
	v.b.WriteByte(']')
}

// VisitTable emits a native table.
func (v *visitor) VisitTable(tn *ast.TableNode) {
	v.b.WriteString("[Table")
	v.level++
	if len(tn.Header) > 0 {
		v.writeNewLine()
		v.b.WriteString("[Header ")
		for i, cell := range tn.Header {
			if i > 0 {
				v.b.WriteByte(',')
			}
			v.writeCell(cell)
		}
		v.b.WriteString("],")
	}
	for i, row := range tn.Rows {
		if i > 0 {
			v.b.WriteByte(',')
		}
		v.writeNewLine()
		v.b.WriteString("[Row ")
		for j, cell := range row {
			if j > 0 {
				v.b.WriteByte(',')
			}
			v.writeCell(cell)
		}
		v.b.WriteByte(']')
	}
	v.level--
	v.b.WriteByte(']')
}

var alignString = map[ast.Alignment]string{
	ast.AlignDefault: " Default",
	ast.AlignLeft:    " Left",
	ast.AlignCenter:  " Center",
	ast.AlignRight:   " Right",
}

func (v *visitor) writeCell(cell *ast.TableCell) {
	v.b.WriteStrings("[Cell", alignString[cell.Align])
	if len(cell.Inlines) > 0 {
		v.b.WriteByte(' ')
		v.acceptInlineSlice(cell.Inlines)
	}
	v.b.WriteByte(']')
}

// VisitBLOB writes the binary object as a value.
func (v *visitor) VisitBLOB(bn *ast.BLOBNode) {
	v.b.WriteString("[BLOB \"")
	v.writeEscaped(bn.Title)
	v.b.WriteString("\" \"")
	v.writeEscaped(bn.Syntax)
	v.b.WriteString("\" \"")
	v.b.WriteBase64(bn.Blob)
	v.b.WriteString("\"]")
}

// VisitText writes text content.
func (v *visitor) VisitText(tn *ast.TextNode) {
	v.b.WriteString("Text \"")
	v.writeEscaped(tn.Text)
	v.b.WriteByte('"')
}

// VisitTag writes tag content.
func (v *visitor) VisitTag(tn *ast.TagNode) {
	v.b.WriteString("Tag \"")
	v.writeEscaped(tn.Tag)
	v.b.WriteByte('"')
}

// VisitSpace emits a white space.
func (v *visitor) VisitSpace(sn *ast.SpaceNode) {
	v.b.WriteString("Space")
	if l := len(sn.Lexeme); l > 1 {
		v.b.WriteByte(' ')
		v.b.WriteString(strconv.Itoa(l))
	}
}

// VisitBreak writes native code for line breaks.
func (v *visitor) VisitBreak(bn *ast.BreakNode) {
	if bn.Hard {
		v.b.WriteString("Break")
	} else {
		v.b.WriteString("Space")
	}
}

var mapRefState = map[ast.RefState]string{
	ast.RefStateInvalid:  "INVALID",
	ast.RefStateZettel:   "ZETTEL",
	ast.RefStateSelf:     "SELF",
	ast.RefStateFound:    "ZETTEL",
	ast.RefStateBroken:   "BROKEN",
	ast.RefStateHosted:   "LOCAL",
	ast.RefStateBased:    "BASED",
	ast.RefStateExternal: "EXTERNAL",
}

// VisitLink writes native code for links.
func (v *visitor) VisitLink(ln *ast.LinkNode) {
	ln, n := v.env.AdaptLink(ln)
	if n != nil {
		n.Accept(v)
		return
	}
	v.b.WriteString("Link")
	v.visitAttributes(ln.Attrs)
	v.b.WriteByte(' ')
	v.b.WriteString(mapRefState[ln.Ref.State])
	v.b.WriteString(" \"")
	v.writeEscaped(ln.Ref.String())
	v.b.WriteString("\" [")
	if !ln.OnlyRef {
		v.acceptInlineSlice(ln.Inlines)
	}
	v.b.WriteByte(']')
}

// VisitImage writes native code for images.
func (v *visitor) VisitImage(in *ast.ImageNode) {
	in, n := v.env.AdaptImage(in)
	if n != nil {
		n.Accept(v)
		return
	}
	v.b.WriteString("Image")
	v.visitAttributes(in.Attrs)
	if in.Ref == nil {
		v.b.WriteStrings(" {\"", in.Syntax, "\" \"")
		switch in.Syntax {
		case "svg":
			v.writeEscaped(string(in.Blob))
		default:
			v.b.WriteString("\" \"")
			v.b.WriteBase64(in.Blob)
		}
		v.b.WriteString("\"}")
	} else {
		v.b.WriteStrings(" \"", in.Ref.String(), "\"")
	}
	if len(in.Inlines) > 0 {
		v.b.WriteString(" [")
		v.acceptInlineSlice(in.Inlines)
		v.b.WriteByte(']')
	}
}

// VisitCite writes code for citations.
func (v *visitor) VisitCite(cn *ast.CiteNode) {
	v.b.WriteString("Cite")
	v.visitAttributes(cn.Attrs)
	v.b.WriteString(" \"")
	v.writeEscaped(cn.Key)
	v.b.WriteByte('"')
	if len(cn.Inlines) > 0 {
		v.b.WriteString(" [")
		v.acceptInlineSlice(cn.Inlines)
		v.b.WriteByte(']')
	}
}

// VisitFootnote write native code for a footnote.
func (v *visitor) VisitFootnote(fn *ast.FootnoteNode) {
	v.b.WriteString("Footnote")
	v.visitAttributes(fn.Attrs)
	v.b.WriteString(" [")
	v.acceptInlineSlice(fn.Inlines)
	v.b.WriteByte(']')
}

// VisitMark writes native code to mark a position.
func (v *visitor) VisitMark(mn *ast.MarkNode) {
	v.b.WriteString("Mark")
	if len(mn.Text) > 0 {
		v.b.WriteString(" \"")
		v.writeEscaped(mn.Text)
		v.b.WriteByte('"')
	}
}

var formatCode = map[ast.FormatCode][]byte{
	ast.FormatItalic:    []byte("Italic"),
	ast.FormatEmph:      []byte("Emph"),
	ast.FormatBold:      []byte("Bold"),
	ast.FormatStrong:    []byte("Strong"),
	ast.FormatUnder:     []byte("Underline"),
	ast.FormatInsert:    []byte("Insert"),
	ast.FormatMonospace: []byte("Mono"),
	ast.FormatStrike:    []byte("Strikethrough"),
	ast.FormatDelete:    []byte("Delete"),
	ast.FormatSuper:     []byte("Super"),
	ast.FormatSub:       []byte("Sub"),
	ast.FormatQuote:     []byte("Quote"),
	ast.FormatQuotation: []byte("Quotation"),
	ast.FormatSmall:     []byte("Small"),
	ast.FormatSpan:      []byte("Span"),
}

// VisitFormat write native code for formatting text.
func (v *visitor) VisitFormat(fn *ast.FormatNode) {
	v.b.Write(formatCode[fn.Code])
	v.visitAttributes(fn.Attrs)
	v.b.WriteString(" [")
	v.acceptInlineSlice(fn.Inlines)
	v.b.WriteByte(']')
}

var literalCode = map[ast.LiteralCode][]byte{
	ast.LiteralProg:    []byte("Code"),
	ast.LiteralKeyb:    []byte("Input"),
	ast.LiteralOutput:  []byte("Output"),
	ast.LiteralComment: []byte("Comment"),
	ast.LiteralHTML:    []byte("HTML"),
}

// VisitLiteral write native code for code inline text.
func (v *visitor) VisitLiteral(ln *ast.LiteralNode) {
	code, ok := literalCode[ln.Code]
	if !ok {
		panic(fmt.Sprintf("Unknown literal code %v", ln.Code))
	}
	v.b.Write(code)
	v.visitAttributes(ln.Attrs)
	v.b.WriteString(" \"")
	v.writeEscaped(ln.Text)
	v.b.WriteByte('"')
}

func (v *visitor) acceptBlockSlice(bns ast.BlockSlice) {
	for i, bn := range bns {
		if i > 0 {
			v.b.WriteByte(',')
			v.writeNewLine()
		}
		bn.Accept(v)
	}
}
func (v *visitor) acceptItemSlice(ins ast.ItemSlice) {
	for i, in := range ins {
		if i > 0 {
			v.b.WriteByte(',')
			v.writeNewLine()
		}
		in.Accept(v)
	}
}
func (v *visitor) acceptDescriptionSlice(dns ast.DescriptionSlice) {
	for i, dn := range dns {
		if i > 0 {
			v.b.WriteByte(',')
			v.writeNewLine()
		}
		dn.Accept(v)
	}
}
func (v *visitor) acceptInlineSlice(ins ast.InlineSlice) {
	for i, in := range ins {
		if i > 0 {
			v.b.WriteByte(',')
		}
		in.Accept(v)
	}
}

// visitAttributes write native attributes
func (v *visitor) visitAttributes(a *ast.Attributes) {
	if a == nil || len(a.Attrs) == 0 {
		return
	}
	keys := make([]string, 0, len(a.Attrs))
	for k := range a.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	v.b.WriteString(" (\"")
	if val, ok := a.Attrs[""]; ok {
		v.writeEscaped(val)
	}
	v.b.WriteString("\",[")
	first := true
	for _, k := range keys {
		if k == "" {
			continue
		}
		if !first {
			v.b.WriteByte(',')
		}
		v.b.WriteString(k)
		val := a.Attrs[k]
		if len(val) > 0 {
			v.b.WriteString("=\"")
			v.writeEscaped(val)
			v.b.WriteByte('"')
		}
		first = false
	}
	v.b.WriteString("])")
}

func (v *visitor) writeNewLine() {
	v.b.WriteByte('\n')
	for i := 0; i < v.level; i++ {
		v.b.WriteByte(' ')
	}
}

func (v *visitor) writeEscaped(s string) {
	last := 0
	for i, ch := range s {
		var b []byte
		switch ch {
		case '\n':
			b = rawNewline
		case '"':
			b = rawDoubleQuote
		case '\\':
			b = rawBackslash
		default:
			continue
		}
		v.b.WriteString(s[last:i])
		v.b.Write(b)
		last = i + 1
	}
	v.b.WriteString(s[last:])
}
