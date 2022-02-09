//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
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

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
)

func init() {
	encoder.Register(api.EncoderNative, encoder.Info{
		Create: func(env *encoder.Environment) encoder.Encoder { return &nativeEncoder{env: env} },
	})
}

type nativeEncoder struct {
	env *encoder.Environment
}

// WriteZettel encodes the zettel to the writer.
func (ne *nativeEncoder) WriteZettel(w io.Writer, zn *ast.ZettelNode, evalMeta encoder.EvalMetaFunc) (int, error) {
	v := newVisitor(w, ne)
	v.acceptMeta(zn.InhMeta, evalMeta)
	v.b.WriteByte('\n')
	ast.Walk(v, zn.Ast)
	length, err := v.b.Flush()
	return length, err
}

// WriteMeta encodes meta data in native format.
func (ne *nativeEncoder) WriteMeta(w io.Writer, m *meta.Meta, evalMeta encoder.EvalMetaFunc) (int, error) {
	v := newVisitor(w, ne)
	v.acceptMeta(m, evalMeta)
	length, err := v.b.Flush()
	return length, err
}

func (ne *nativeEncoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return ne.WriteBlocks(w, zn.Ast)
}

// WriteBlocks writes a block slice to the writer
func (ne *nativeEncoder) WriteBlocks(w io.Writer, bln *ast.BlockListNode) (int, error) {
	v := newVisitor(w, ne)
	ast.Walk(v, bln)
	length, err := v.b.Flush()
	return length, err
}

// WriteInlines writes an inline slice to the writer
func (ne *nativeEncoder) WriteInlines(w io.Writer, iln *ast.InlineListNode) (int, error) {
	v := newVisitor(w, ne)
	ast.Walk(v, iln)
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

func (v *visitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.BlockListNode:
		v.visitBlockList(n)
	case *ast.InlineListNode:
		v.walkInlineList(n)
	case *ast.ParaNode:
		v.b.WriteString("[Para ")
		ast.Walk(v, n.Inlines)
		v.b.WriteByte(']')
	case *ast.VerbatimNode:
		v.visitVerbatim(n)
	case *ast.RegionNode:
		v.visitRegion(n)
	case *ast.HeadingNode:
		v.visitHeading(n)
	case *ast.HRuleNode:
		v.b.WriteString("[Hrule")
		v.visitAttributes(n.Attrs)
		v.b.WriteByte(']')
	case *ast.NestedListNode:
		v.visitNestedList(n)
	case *ast.DescriptionListNode:
		v.visitDescriptionList(n)
	case *ast.TableNode:
		v.visitTable(n)
	case *ast.TranscludeNode:
		v.b.WriteString("[Transclude ")
		v.b.WriteString(mapRefState[n.Ref.State])
		v.b.WriteString(" \"")
		v.writeEscaped(n.Ref.String())
		v.b.WriteString("\"]")
	case *ast.BLOBNode:
		v.visitBLOB(n)
	case *ast.TextNode:
		v.b.WriteString("Text \"")
		v.writeEscaped(n.Text)
		v.b.WriteByte('"')
	case *ast.TagNode:
		v.b.WriteString("Tag \"")
		v.writeEscaped(n.Tag)
		v.b.WriteByte('"')
	case *ast.SpaceNode:
		v.b.WriteString("Space")
		if l := len(n.Lexeme); l > 1 {
			v.b.WriteByte(' ')
			v.b.WriteString(strconv.Itoa(l))
		}
	case *ast.BreakNode:
		if n.Hard {
			v.b.WriteString("Break")
		} else {
			v.b.WriteString("Space")
		}
	case *ast.LinkNode:
		v.visitLink(n)
	case *ast.EmbedRefNode:
		v.visitEmbedRef(n)
	case *ast.EmbedBLOBNode:
		v.visitEmbedBLOB(n)
	case *ast.CiteNode:
		v.b.WriteString("Cite")
		v.visitAttributes(n.Attrs)
		v.b.WriteString(" \"")
		v.writeEscaped(n.Key)
		v.b.WriteByte('"')
		if n.Inlines != nil {
			v.b.WriteString(" [")
			ast.Walk(v, n.Inlines)
			v.b.WriteByte(']')
		}
	case *ast.FootnoteNode:
		v.b.WriteString("Footnote")
		v.visitAttributes(n.Attrs)
		v.b.WriteString(" [")
		ast.Walk(v, n.Inlines)
		v.b.WriteByte(']')
	case *ast.MarkNode:
		v.visitMark(n)
	case *ast.FormatNode:
		v.b.Write(mapFormatKind[n.Kind])
		v.visitAttributes(n.Attrs)
		v.b.WriteString(" [")
		ast.Walk(v, n.Inlines)
		v.b.WriteByte(']')
	case *ast.LiteralNode:
		kind, ok := mapLiteralKind[n.Kind]
		if !ok {
			panic(fmt.Sprintf("Unknown literal kind %v", n.Kind))
		}
		v.b.Write(kind)
		v.visitAttributes(n.Attrs)
		v.b.WriteString(" \"")
		v.writeEscaped(string(n.Content))
		v.b.WriteByte('"')
	default:
		return v
	}
	return nil
}

var (
	rawBackslash   = []byte{'\\', '\\'}
	rawDoubleQuote = []byte{'\\', '"'}
	rawNewline     = []byte{'\\', 'n'}
)

func (v *visitor) acceptMeta(m *meta.Meta, evalMeta encoder.EvalMetaFunc) {
	v.writeZettelmarkup("Title", m.GetDefault(api.KeyTitle, ""), evalMeta)
	v.writeMetaString(m, api.KeyRole, "Role")
	v.writeMetaList(m, api.KeyTags, "Tags")
	v.writeMetaString(m, api.KeySyntax, "Syntax")
	pairs := m.ComputedPairsRest()
	if len(pairs) == 0 {
		return
	}
	v.b.WriteString("\n[Header")
	v.level++
	for i, p := range pairs {
		v.writeComma(i)
		v.writeNewLine()
		key, value := p.Key, p.Value
		if meta.Type(key) == meta.TypeZettelmarkup {
			v.writeZettelmarkup(key, value, evalMeta)
		} else {
			v.b.WriteByte('[')
			v.b.WriteStrings(key, " \"")
			v.writeEscaped(value)
			v.b.WriteString("\"]")
		}
	}
	v.level--
	v.b.WriteByte(']')
}

func (v *visitor) writeZettelmarkup(key, value string, evalMeta encoder.EvalMetaFunc) {
	v.b.WriteByte('[')
	v.b.WriteString(key)
	v.b.WriteByte(' ')
	ast.Walk(v, evalMeta(value))
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

var mapVerbatimKind = map[ast.VerbatimKind][]byte{
	ast.VerbatimZettel:  []byte("[ZettelBlock"),
	ast.VerbatimProg:    []byte("[CodeBlock"),
	ast.VerbatimComment: []byte("[CommentBlock"),
	ast.VerbatimHTML:    []byte("[HTMLBlock"),
}

func (v *visitor) visitVerbatim(vn *ast.VerbatimNode) {
	kind, ok := mapVerbatimKind[vn.Kind]
	if !ok {
		panic(fmt.Sprintf("Unknown verbatim kind %v", vn.Kind))
	}
	v.b.Write(kind)
	v.visitAttributes(vn.Attrs)
	v.b.WriteString(" \"")
	v.writeEscaped(string(vn.Content))
	v.b.WriteString("\"]")
}

var mapRegionKind = map[ast.RegionKind][]byte{
	ast.RegionSpan:  []byte("[SpanBlock"),
	ast.RegionQuote: []byte("[QuoteBlock"),
	ast.RegionVerse: []byte("[VerseBlock"),
}

func (v *visitor) visitRegion(rn *ast.RegionNode) {
	kind, ok := mapRegionKind[rn.Kind]
	if !ok {
		panic(fmt.Sprintf("Unknown region kind %v", rn.Kind))
	}
	v.b.Write(kind)
	v.visitAttributes(rn.Attrs)
	v.level++
	v.writeNewLine()
	v.b.WriteByte('[')
	v.level++
	ast.Walk(v, rn.Blocks)
	v.level--
	v.b.WriteByte(']')
	if rn.Inlines != nil {
		v.b.WriteByte(',')
		v.writeNewLine()
		v.b.WriteString("[Cite ")
		ast.Walk(v, rn.Inlines)
		v.b.WriteByte(']')
	}
	v.level--
	v.b.WriteByte(']')
}

func (v *visitor) visitHeading(hn *ast.HeadingNode) {
	v.b.WriteStrings("[Heading ", strconv.Itoa(hn.Level))
	if fragment := hn.Fragment; fragment != "" {
		v.b.WriteStrings(" #", fragment)
	}
	v.visitAttributes(hn.Attrs)
	v.b.WriteByte(' ')
	ast.Walk(v, hn.Inlines)
	v.b.WriteByte(']')
}

var mapNestedListKind = map[ast.NestedListKind][]byte{
	ast.NestedListOrdered:   []byte("[OrderedList"),
	ast.NestedListUnordered: []byte("[BulletList"),
	ast.NestedListQuote:     []byte("[QuoteList"),
}

func (v *visitor) visitNestedList(ln *ast.NestedListNode) {
	v.b.Write(mapNestedListKind[ln.Kind])
	v.level++
	for i, item := range ln.Items {
		v.writeComma(i)
		v.writeNewLine()
		v.level++
		v.b.WriteByte('[')
		for i, in := range item {
			if i > 0 {
				v.b.WriteByte(',')
				v.writeNewLine()
			}
			ast.Walk(v, in)
		}
		v.b.WriteByte(']')
		v.level--
	}
	v.level--
	v.b.WriteByte(']')
}

func (v *visitor) visitDescriptionList(dn *ast.DescriptionListNode) {
	v.b.WriteString("[DescriptionList")
	v.level++
	for i, descr := range dn.Descriptions {
		v.writeComma(i)
		v.writeNewLine()
		v.b.WriteString("[Term [")
		ast.Walk(v, descr.Term)
		v.b.WriteByte(']')

		if len(descr.Descriptions) > 0 {
			v.level++
			for _, b := range descr.Descriptions {
				v.b.WriteByte(',')
				v.writeNewLine()
				v.b.WriteString("[Description")
				v.level++
				v.writeNewLine()
				for i, dn := range b {
					if i > 0 {
						v.b.WriteByte(',')
						v.writeNewLine()
					}
					ast.Walk(v, dn)
				}
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

func (v *visitor) visitTable(tn *ast.TableNode) {
	v.b.WriteString("[Table")
	v.level++
	if len(tn.Header) > 0 {
		v.writeNewLine()
		v.b.WriteString("[Header ")
		for i, cell := range tn.Header {
			v.writeComma(i)
			v.writeCell(cell)
		}
		v.b.WriteString("],")
	}
	for i, row := range tn.Rows {
		v.writeComma(i)
		v.writeNewLine()
		v.b.WriteString("[Row ")
		for j, cell := range row {
			v.writeComma(j)
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
	if !cell.Inlines.IsEmpty() {
		v.b.WriteByte(' ')
		ast.Walk(v, cell.Inlines)
	}
	v.b.WriteByte(']')
}

func (v *visitor) visitBLOB(bn *ast.BLOBNode) {
	v.b.WriteString("[BLOB \"")
	v.writeEscaped(bn.Title)
	v.b.WriteString("\" \"")
	v.writeEscaped(bn.Syntax)
	v.b.WriteString("\" \"")
	if bn.Syntax == api.ValueSyntaxSVG {
		v.writeEscaped(string(bn.Blob))
	} else {
		v.b.WriteBase64(bn.Blob)
	}
	v.b.WriteString("\"]")
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

func (v *visitor) visitLink(ln *ast.LinkNode) {
	v.b.WriteString("Link")
	v.visitAttributes(ln.Attrs)
	v.b.WriteByte(' ')
	v.b.WriteString(mapRefState[ln.Ref.State])
	v.b.WriteString(" \"")
	v.writeEscaped(ln.Ref.String())
	v.b.WriteString("\" [")
	if !ln.OnlyRef {
		ast.Walk(v, ln.Inlines)
	}
	v.b.WriteByte(']')
}

func (v *visitor) visitEmbedRef(en *ast.EmbedRefNode) {
	v.b.WriteString("Embed")
	v.visitAttributes(en.Attrs)
	v.b.WriteByte(' ')
	v.b.WriteString(mapRefState[en.Ref.State])
	v.b.WriteString(" \"")
	v.writeEscaped(en.Ref.String())
	v.b.WriteByte('"')

	if en.Inlines != nil {
		v.b.WriteString(" [")
		ast.Walk(v, en.Inlines)
		v.b.WriteByte(']')
	}
}

func (v *visitor) visitEmbedBLOB(en *ast.EmbedBLOBNode) {
	v.b.WriteString("EmbedBLOB")
	v.visitAttributes(en.Attrs)
	v.b.WriteStrings(" {\"", en.Syntax, "\" \"")
	if en.Syntax == api.ValueSyntaxSVG {
		v.writeEscaped(string(en.Blob))
	} else {
		v.b.WriteString("\" \"")
		v.b.WriteBase64(en.Blob)
	}
	v.b.WriteString("\"}")

	if en.Inlines != nil {
		v.b.WriteString(" [")
		ast.Walk(v, en.Inlines)
		v.b.WriteByte(']')
	}
}

func (v *visitor) visitMark(mn *ast.MarkNode) {
	v.b.WriteString("Mark")
	if text := mn.Text; text != "" {
		v.b.WriteString(" \"")
		v.writeEscaped(text)
		v.b.WriteByte('"')
	}
	if fragment := mn.Fragment; fragment != "" {
		v.b.WriteString(" #")
		v.writeEscaped(fragment)
	}
}

var mapFormatKind = map[ast.FormatKind][]byte{
	ast.FormatEmph:      []byte("Emph"),
	ast.FormatStrong:    []byte("Strong"),
	ast.FormatInsert:    []byte("Insert"),
	ast.FormatMonospace: []byte("Mono"),
	ast.FormatDelete:    []byte("Delete"),
	ast.FormatSuper:     []byte("Super"),
	ast.FormatSub:       []byte("Sub"),
	ast.FormatQuote:     []byte("Quote"),
	ast.FormatQuotation: []byte("Quotation"),
	ast.FormatSpan:      []byte("Span"),
}

var mapLiteralKind = map[ast.LiteralKind][]byte{
	ast.LiteralZettel:  []byte("Zettel"),
	ast.LiteralProg:    []byte("Code"),
	ast.LiteralKeyb:    []byte("Input"),
	ast.LiteralOutput:  []byte("Output"),
	ast.LiteralComment: []byte("Comment"),
	ast.LiteralHTML:    []byte("HTML"),
}

func (v *visitor) visitBlockList(bln *ast.BlockListNode) {
	for i, bn := range bln.List {
		if i > 0 {
			v.b.WriteByte(',')
			v.writeNewLine()
		}
		ast.Walk(v, bn)
	}
}
func (v *visitor) walkInlineList(iln *ast.InlineListNode) {
	for i, in := range iln.List {
		v.writeComma(i)
		ast.Walk(v, in)
	}
}

// visitAttributes write native attributes
func (v *visitor) visitAttributes(a ast.Attributes) {
	if a.IsEmpty() {
		return
	}
	keys := make([]string, 0, len(a))
	for k := range a {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	v.b.WriteString(" (\"")
	if val, ok := a[""]; ok {
		v.writeEscaped(val)
	}
	v.b.WriteString("\",[")
	for i, k := range keys {
		if k == "" {
			continue
		}
		v.writeComma(i)
		v.b.WriteString(k)
		val := a[k]
		if len(val) > 0 {
			v.b.WriteString("=\"")
			v.writeEscaped(val)
			v.b.WriteByte('"')
		}
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

func (v *visitor) writeComma(pos int) {
	if pos > 0 {
		v.b.WriteByte(',')
	}
}
