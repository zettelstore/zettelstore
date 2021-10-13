//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package zmkenc encodes the abstract syntax tree back into Zettelmarkup.
package zmkenc

import (
	"fmt"
	"io"
	"sort"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
)

func init() {
	encoder.Register(api.EncoderZmk, encoder.Info{
		Create: func(*encoder.Environment) encoder.Encoder { return &zmkEncoder{} },
	})
}

type zmkEncoder struct{}

// WriteZettel writes the encoded zettel to the writer.
func (ze *zmkEncoder) WriteZettel(w io.Writer, zn *ast.ZettelNode, evalMeta encoder.EvalMetaFunc) (int, error) {
	v := newVisitor(w, ze)
	v.acceptMeta(zn.InhMeta, evalMeta)
	if zn.InhMeta.YamlSep {
		v.b.WriteString("---\n")
	} else {
		v.b.WriteByte('\n')
	}
	ast.Walk(v, zn.Ast)
	length, err := v.b.Flush()
	return length, err
}

// WriteMeta encodes meta data as zmk.
func (ze *zmkEncoder) WriteMeta(w io.Writer, m *meta.Meta, evalMeta encoder.EvalMetaFunc) (int, error) {
	v := newVisitor(w, ze)
	v.acceptMeta(m, evalMeta)
	length, err := v.b.Flush()
	return length, err
}

func (v *visitor) acceptMeta(m *meta.Meta, evalMeta encoder.EvalMetaFunc) {
	for _, p := range m.Pairs(true) {
		key := p.Key
		v.b.WriteStrings(key, ": ")
		if meta.Type(key) == meta.TypeZettelmarkup {
			ast.Walk(v, evalMeta(p.Value))
		} else {
			v.b.WriteString(p.Value)
		}
		v.b.WriteByte('\n')
	}
}

func (ze *zmkEncoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return ze.WriteBlocks(w, zn.Ast)
}

// WriteBlocks writes the content of a block slice to the writer.
func (ze *zmkEncoder) WriteBlocks(w io.Writer, bln *ast.BlockListNode) (int, error) {
	v := newVisitor(w, ze)
	ast.Walk(v, bln)
	length, err := v.b.Flush()
	return length, err
}

// WriteInlines writes an inline slice to the writer
func (ze *zmkEncoder) WriteInlines(w io.Writer, iln *ast.InlineListNode) (int, error) {
	v := newVisitor(w, ze)
	ast.Walk(v, iln)
	length, err := v.b.Flush()
	return length, err
}

// visitor writes the abstract syntax tree to an io.Writer.
type visitor struct {
	b      encoder.BufWriter
	prefix []byte
	enc    *zmkEncoder
}

func newVisitor(w io.Writer, enc *zmkEncoder) *visitor {
	return &visitor{
		b:   encoder.NewBufWriter(w),
		enc: enc,
	}
}

func (v *visitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.BlockListNode:
		for i, bn := range n.List {
			if i > 0 {
				v.b.WriteByte('\n')
			}
			ast.Walk(v, bn)
		}
	case *ast.VerbatimNode:
		v.visitVerbatim(n)
	case *ast.RegionNode:
		v.visitRegion(n)
	case *ast.HeadingNode:
		v.visitHeading(n)
	case *ast.HRuleNode:
		v.b.WriteString("---")
		v.visitAttributes(n.Attrs)
	case *ast.NestedListNode:
		v.visitNestedList(n)
	case *ast.DescriptionListNode:
		v.visitDescriptionList(n)
	case *ast.TableNode:
		v.visitTable(n)
	case *ast.BLOBNode:
		v.b.WriteStrings(
			"%% Unable to display BLOB with title '", n.Title,
			"' and syntax '", n.Syntax, "'.")
	case *ast.TextNode:
		v.visitText(n)
	case *ast.TagNode:
		v.b.WriteStrings("#", n.Tag)
	case *ast.SpaceNode:
		v.b.WriteString(n.Lexeme)
	case *ast.BreakNode:
		v.visitBreak(n)
	case *ast.LinkNode:
		v.visitLink(n)
	case *ast.EmbedNode:
		v.visitEmbed(n)
	case *ast.CiteNode:
		v.visitCite(n)
	case *ast.FootnoteNode:
		v.b.WriteString("[^")
		ast.Walk(v, n.Inlines)
		v.b.WriteByte(']')
		v.visitAttributes(n.Attrs)
	case *ast.MarkNode:
		v.b.WriteStrings("[!", n.Text, "]")
	case *ast.FormatNode:
		v.visitFormat(n)
	case *ast.LiteralNode:
		v.visitLiteral(n)
	default:
		return v
	}
	return nil
}

var mapVerbatimKind = map[ast.VerbatimKind]string{
	ast.VerbatimComment: "%%%",
	ast.VerbatimHTML:    "???",
	ast.VerbatimProg:    "```",
}

func (v *visitor) visitVerbatim(vn *ast.VerbatimNode) {
	kind, ok := mapVerbatimKind[vn.Kind]
	if !ok {
		panic(fmt.Sprintf("Unknown verbatim kind %d", vn.Kind))
	}

	// TODO: scan cn.Lines to find embedded "`"s at beginning
	v.b.WriteString(kind)
	v.visitAttributes(vn.Attrs)
	v.b.WriteByte('\n')
	for _, line := range vn.Lines {
		v.b.WriteStrings(line, "\n")
	}
	v.b.WriteString(kind)
}

var mapRegionKind = map[ast.RegionKind]string{
	ast.RegionSpan:  ":::",
	ast.RegionQuote: "<<<",
	ast.RegionVerse: "\"\"\"",
}

func (v *visitor) visitRegion(rn *ast.RegionNode) {
	// Scan rn.Blocks for embedded regions to adjust length of regionCode
	kind, ok := mapRegionKind[rn.Kind]
	if !ok {
		panic(fmt.Sprintf("Unknown region kind %d", rn.Kind))
	}
	v.b.WriteString(kind)
	v.visitAttributes(rn.Attrs)
	v.b.WriteByte('\n')
	ast.Walk(v, rn.Blocks)
	v.b.WriteByte('\n')
	v.b.WriteString(kind)
	if rn.Inlines != nil {
		v.b.WriteByte(' ')
		ast.Walk(v, rn.Inlines)
	}
}

func (v *visitor) visitHeading(hn *ast.HeadingNode) {
	for i := 0; i <= hn.Level; i++ {
		v.b.WriteByte('=')
	}
	v.b.WriteByte(' ')
	ast.Walk(v, hn.Inlines)
	v.visitAttributes(hn.Attrs)
}

var mapNestedListKind = map[ast.NestedListKind]byte{
	ast.NestedListOrdered:   '#',
	ast.NestedListUnordered: '*',
	ast.NestedListQuote:     '>',
}

func (v *visitor) visitNestedList(ln *ast.NestedListNode) {
	v.prefix = append(v.prefix, mapNestedListKind[ln.Kind])
	for i, item := range ln.Items {
		if i > 0 {
			v.b.WriteByte('\n')
		}
		v.b.Write(v.prefix)
		v.b.WriteByte(' ')
		for i, in := range item {
			if i > 0 {
				v.b.WriteByte('\n')
				if _, ok := in.(*ast.ParaNode); ok {
					for j := 0; j <= len(v.prefix); j++ {
						v.b.WriteByte(' ')
					}
				}
			}
			ast.Walk(v, in)
		}
	}
	v.prefix = v.prefix[:len(v.prefix)-1]
}

func (v *visitor) visitDescriptionList(dn *ast.DescriptionListNode) {
	for i, descr := range dn.Descriptions {
		if i > 0 {
			v.b.WriteByte('\n')
		}
		v.b.WriteString("; ")
		ast.Walk(v, descr.Term)

		for _, b := range descr.Descriptions {
			v.b.WriteString("\n: ")
			ast.WalkDescriptionSlice(v, b)
		}
	}
}

var alignCode = map[ast.Alignment]string{
	ast.AlignDefault: "",
	ast.AlignLeft:    "<",
	ast.AlignCenter:  ":",
	ast.AlignRight:   ">",
}

func (v *visitor) visitTable(tn *ast.TableNode) {
	if len(tn.Header) > 0 {
		for pos, cell := range tn.Header {
			v.b.WriteString("|=")
			colAlign := tn.Align[pos]
			if cell.Align != colAlign {
				v.b.WriteString(alignCode[cell.Align])
			}
			ast.Walk(v, cell.Inlines)
			if colAlign != ast.AlignDefault {
				v.b.WriteString(alignCode[colAlign])
			}
		}
		v.b.WriteByte('\n')
	}
	for i, row := range tn.Rows {
		if i > 0 {
			v.b.WriteByte('\n')
		}
		for pos, cell := range row {
			v.b.WriteByte('|')
			if cell.Align != tn.Align[pos] {
				v.b.WriteString(alignCode[cell.Align])
			}
			ast.Walk(v, cell.Inlines)
		}
	}
}

var escapeSeqs = map[string]bool{
	"\\":   true,
	"//":   true,
	"**":   true,
	"__":   true,
	"~~":   true,
	"^^":   true,
	",,":   true,
	"<<":   true,
	"\"\"": true,
	";;":   true,
	"::":   true,
	"''":   true,
	"``":   true,
	"++":   true,
	"==":   true,
}

func (v *visitor) visitText(tn *ast.TextNode) {
	last := 0
	for i := 0; i < len(tn.Text); i++ {
		if b := tn.Text[i]; b == '\\' {
			v.b.WriteString(tn.Text[last:i])
			v.b.WriteBytes('\\', b)
			last = i + 1
			continue
		}
		if i < len(tn.Text)-1 {
			s := tn.Text[i : i+2]
			if _, ok := escapeSeqs[s]; ok {
				v.b.WriteString(tn.Text[last:i])
				for j := 0; j < len(s); j++ {
					v.b.WriteBytes('\\', s[j])
				}
				i++
				last = i + 1
				continue
			}
		}
	}
	v.b.WriteString(tn.Text[last:])
}

func (v *visitor) visitBreak(bn *ast.BreakNode) {
	if bn.Hard {
		v.b.WriteString("\\\n")
	} else {
		v.b.WriteByte('\n')
	}
	if prefixLen := len(v.prefix); prefixLen > 0 {
		for i := 0; i <= prefixLen; i++ {
			v.b.WriteByte(' ')
		}
	}
}

func (v *visitor) visitLink(ln *ast.LinkNode) {
	v.b.WriteString("[[")
	if !ln.OnlyRef {
		ast.Walk(v, ln.Inlines)
		v.b.WriteByte('|')
	}
	v.b.WriteStrings(ln.Ref.String(), "]]")
}

func (v *visitor) visitEmbed(en *ast.EmbedNode) {
	switch m := en.Material.(type) {
	case *ast.ReferenceMaterialNode:
		v.b.WriteString("{{")
		if en.Inlines != nil {
			ast.Walk(v, en.Inlines)
			v.b.WriteByte('|')
		}
		v.b.WriteStrings(m.Ref.String(), "}}")
	case *ast.BLOBMaterialNode:
		panic("TODO")
	default:
		panic(fmt.Sprintf("Unknown material type %t for %v", en.Material, en.Material))
	}
}

func (v *visitor) visitCite(cn *ast.CiteNode) {
	v.b.WriteStrings("[@", cn.Key)
	if cn.Inlines != nil {
		v.b.WriteString(", ")
		ast.Walk(v, cn.Inlines)
	}
	v.b.WriteByte(']')
	v.visitAttributes(cn.Attrs)
}

var mapFormatKind = map[ast.FormatKind][]byte{
	ast.FormatItalic:    []byte("//"),
	ast.FormatEmph:      []byte("//"),
	ast.FormatBold:      []byte("**"),
	ast.FormatStrong:    []byte("**"),
	ast.FormatUnder:     []byte("__"),
	ast.FormatInsert:    []byte("__"),
	ast.FormatStrike:    []byte("~~"),
	ast.FormatDelete:    []byte("~~"),
	ast.FormatSuper:     []byte("^^"),
	ast.FormatSub:       []byte(",,"),
	ast.FormatQuotation: []byte("<<"),
	ast.FormatQuote:     []byte("\"\""),
	ast.FormatSmall:     []byte(";;"),
	ast.FormatSpan:      []byte("::"),
	ast.FormatMonospace: []byte("''"),
}

func (v *visitor) visitFormat(fn *ast.FormatNode) {
	kind, ok := mapFormatKind[fn.Kind]
	if !ok {
		panic(fmt.Sprintf("Unknown format kind %d", fn.Kind))
	}
	attrs := fn.Attrs
	switch fn.Kind {
	case ast.FormatEmph, ast.FormatStrong, ast.FormatInsert, ast.FormatDelete:
		attrs = attrs.Clone()
		attrs.Set("-", "")
	}

	v.b.Write(kind)
	ast.Walk(v, fn.Inlines)
	v.b.Write(kind)
	v.visitAttributes(attrs)
}

func (v *visitor) visitLiteral(ln *ast.LiteralNode) {
	switch ln.Kind {
	case ast.LiteralProg:
		v.writeLiteral('`', ln.Attrs, ln.Text)
	case ast.LiteralKeyb:
		v.writeLiteral('+', ln.Attrs, ln.Text)
	case ast.LiteralOutput:
		v.writeLiteral('=', ln.Attrs, ln.Text)
	case ast.LiteralComment:
		v.b.WriteStrings("%% ", ln.Text)
	case ast.LiteralHTML:
		v.b.WriteString("``")
		v.writeEscaped(ln.Text, '`')
		v.b.WriteString("``{=html,.warning}")
	default:
		panic(fmt.Sprintf("Unknown literal kind %v", ln.Kind))
	}
}

func (v *visitor) writeLiteral(code byte, attrs *ast.Attributes, text string) {
	v.b.WriteBytes(code, code)
	v.writeEscaped(text, code)
	v.b.WriteBytes(code, code)
	v.visitAttributes(attrs)
}

// visitAttributes write HTML attributes
func (v *visitor) visitAttributes(a *ast.Attributes) {
	if a.IsEmpty() {
		return
	}
	keys := make([]string, 0, len(a.Attrs))
	for k := range a.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	v.b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			v.b.WriteByte(' ')
		}
		if k == "-" {
			v.b.WriteByte('-')
			continue
		}
		v.b.WriteString(k)
		if vl := a.Attrs[k]; len(vl) > 0 {
			v.b.WriteStrings("=\"", vl)
			v.b.WriteByte('"')
		}
	}
	v.b.WriteByte('}')
}

func (v *visitor) writeEscaped(s string, toEscape byte) {
	last := 0
	for i := 0; i < len(s); i++ {
		if b := s[i]; b == toEscape || b == '\\' {
			v.b.WriteString(s[last:i])
			v.b.WriteBytes('\\', b)
			last = i + 1
		}
	}
	v.b.WriteString(s[last:])
}
