//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package sexprenc encodes the abstract syntax tree into a s-expr.
package sexprenc

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"strconv"

	"zettelstore.de/c/api"
	"zettelstore.de/c/zjson"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/strfun"
)

func init() {
	encoder.Register(api.EncoderSexpr, func() encoder.Encoder { return Create() })
}

// Create a S-expr encoder
func Create() *Encoder { return &mySE }

type Encoder struct{}

var mySE Encoder

// WriteZettel writes the encoded zettel to the writer.
func (*Encoder) WriteZettel(w io.Writer, zn *ast.ZettelNode, evalMeta encoder.EvalMetaFunc) (int, error) {
	v := visitor{w: w}
	v.WriteOpen()
	v.writeMeta(zn.InhMeta, evalMeta)
	v.WriteSep()
	ast.Walk(&v, &zn.Ast)
	v.WriteClose()
	return v.length, v.err
}

// WriteMeta encodes meta data as JSON.
func (*Encoder) WriteMeta(w io.Writer, m *meta.Meta, evalMeta encoder.EvalMetaFunc) (int, error) {
	v := visitor{w: w}
	v.writeMeta(m, evalMeta)
	return v.length, v.err
}

func (se *Encoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return se.WriteBlocks(w, &zn.Ast)
}

// WriteBlocks writes a block slice to the writer
func (*Encoder) WriteBlocks(w io.Writer, bs *ast.BlockSlice) (int, error) {
	v := visitor{w: w}
	ast.Walk(&v, bs)
	return v.length, v.err
}

// WriteInlines writes an inline slice to the writer
func (*Encoder) WriteInlines(w io.Writer, is *ast.InlineSlice) (int, error) {
	v := visitor{w: w}
	ast.Walk(&v, is)
	return v.length, v.err
}

type visitor struct {
	w       io.Writer
	length  int
	err     error
	inVerse bool
}

func (v *visitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	v.WriteOpen()
	switch n := node.(type) {
	case *ast.BlockSlice:
		v.writeBlockSlice(n)
	case *ast.InlineSlice:
		v.writeInlineSlice(*n)
	case *ast.ParaNode:
		v.WriteSymbol("PARA")
		v.writeOptInlineSlice(n.Inlines)
	case *ast.VerbatimNode:
		v.WriteSymbol(mapGet(mapVerbatimKind, n.Kind))
		v.writeAttributes(n.Attrs)
		v.WriteSep()
		v.WriteEscapedSymbol(string(n.Content))
	case *ast.RegionNode:
		v.visitRegion(n)
	case *ast.HeadingNode:
		v.WriteSymbol("HEADING")
		v.WriteSep()
		v.WriteSymbol(strconv.Itoa(n.Level))
		v.writeAttributes(n.Attrs)
		v.WriteSep()
		v.WriteStringSymbol(n.Slug)
		v.WriteSep()
		v.WriteStringSymbol(n.Fragment)
		v.writeOptInlineSlice(n.Inlines)
	case *ast.HRuleNode:
		v.WriteSymbol("THEMATIC")
		v.writeAttributes(n.Attrs)
	case *ast.NestedListNode:
		v.visitNestedList(n)
	case *ast.DescriptionListNode:
		v.visitDescriptionList(n)
	case *ast.TableNode:
		v.visitTable(n)
	case *ast.TranscludeNode:
		v.WriteSymbol("TRANSCLUDE")
		v.writeReference(n.Ref)
	case *ast.BLOBNode:
		v.visitBLOB(n)
	case *ast.TextNode:
		v.WriteSymbol("TEXT")
		v.WriteSep()
		v.WriteStringSymbol(n.Text)
	case *ast.TagNode:
		v.WriteSymbol("TAG")
		v.WriteSep()
		v.WriteStringSymbol(n.Tag)
	case *ast.SpaceNode:
		v.WriteSymbol("SPACE")
		if v.inVerse {
			v.WriteSep()
			v.WriteEscapedSymbol(n.Lexeme)
		}
	case *ast.BreakNode:
		if n.Hard {
			v.WriteSymbol("HARD")
		} else {
			v.WriteSymbol("SOFT")
		}
	case *ast.LinkNode:
		v.WriteSymbol("LINK")
		v.writeAttributes(n.Attrs)
		v.writeReference(n.Ref)
		v.writeOptInlineSlice(n.Inlines)
	case *ast.EmbedRefNode:
		v.WriteSymbol("EMBED")
		v.writeAttributes(n.Attrs)
		v.writeReference(n.Ref)
		v.WriteSep()
		v.WriteStringSymbol(n.Syntax)
		v.writeOptInlineSlice(n.Inlines)
	case *ast.EmbedBLOBNode:
		v.visitEmbedBLOB(n)
	case *ast.CiteNode:
		v.WriteSymbol("CITE")
		v.writeAttributes(n.Attrs)
		v.WriteSep()
		v.WriteStringSymbol(n.Key)
		v.writeOptInlineSlice(n.Inlines)
	case *ast.FootnoteNode:
		v.WriteSymbol("FOOTNOTE")
		v.writeAttributes(n.Attrs)
		v.writeOptInlineSlice(n.Inlines)
	case *ast.MarkNode:
		v.WriteSymbol("MARK")
		v.WriteSep()
		v.WriteStringSymbol(n.Mark)
		v.WriteSep()
		v.WriteStringSymbol(n.Slug)
		v.WriteSep()
		v.WriteStringSymbol(n.Fragment)
		v.writeOptInlineSlice(n.Inlines)
	case *ast.FormatNode:
		v.WriteSymbol(mapGet(mapFormatKind, n.Kind))
		v.writeAttributes(n.Attrs)
		v.writeOptInlineSlice(n.Inlines)
	case *ast.LiteralNode:
		v.WriteSymbol(mapGet(mapLiteralKind, n.Kind))
		v.writeAttributes(n.Attrs)
		v.WriteSep()
		v.WriteEscapedSymbol(string(n.Content))
	default:
		log.Printf("SEXPR %T %v\n", node, node)
		v.WriteSymbol("UNKNOWN-NODE")
		v.WriteSep()
		v.WriteStringSymbol(fmt.Sprintf("%T %v", node, node))
	}
	v.WriteClose()
	return nil
}

var mapVerbatimKind = map[ast.VerbatimKind]string{
	ast.VerbatimZettel:  "VERBATIM-ZETTEL",
	ast.VerbatimProg:    "VERBATIM-CODE",
	ast.VerbatimEval:    "VERBATIM-EVAL",
	ast.VerbatimMath:    "VERBATIM-MATH",
	ast.VerbatimComment: "VERBATIM-COMMENT",
	ast.VerbatimHTML:    "VERBATIM-HTML",
}

var mapRegionKind = map[ast.RegionKind]string{
	ast.RegionSpan:  "REGION-BLOCK",
	ast.RegionQuote: "REGION-QUOTE",
	ast.RegionVerse: "REGION-VERSE",
}

func (v *visitor) visitRegion(rn *ast.RegionNode) {
	saveInVerse := v.inVerse
	if rn.Kind == ast.RegionVerse {
		v.inVerse = true
	}
	v.WriteSymbol(mapGet(mapRegionKind, rn.Kind))
	v.writeAttributes(rn.Attrs)
	v.WriteSep()
	ast.Walk(v, &rn.Blocks)
	v.WriteSep()
	ast.Walk(v, &rn.Inlines)
	v.inVerse = saveInVerse
}

var mapNestedListKind = map[ast.NestedListKind]string{
	ast.NestedListOrdered:   "ORDERED",
	ast.NestedListUnordered: "UNORDERED",
	ast.NestedListQuote:     "QUOTATION",
}

func (v *visitor) visitNestedList(ln *ast.NestedListNode) {
	v.WriteSymbol(mapGet(mapNestedListKind, ln.Kind))
	for _, item := range ln.Items {
		v.WriteSep()
		v.WriteOpen()
		for j, in := range item {
			if j > 0 {
				v.WriteSep()
			}
			ast.Walk(v, in)
		}
		v.WriteClose()
	}
}

func (v *visitor) visitDescriptionList(dn *ast.DescriptionListNode) {
	v.WriteSymbol("DESCRIPTION")
	for _, def := range dn.Descriptions {
		v.WriteSep()
		v.WriteOpen()
		v.writeInlineSlice(def.Term)
		v.WriteClose()

		v.WriteOpen()
		for j, b := range def.Descriptions {
			if j > 0 {
				v.WriteSep()
			}
			v.WriteOpen()
			for k, dn := range b {
				if k > 0 {
					v.WriteSep()
				}
				ast.Walk(v, dn)
			}
			v.WriteClose()
		}
		v.WriteClose()
	}
}

func (v *visitor) visitTable(tn *ast.TableNode) {
	v.WriteSymbol("TABLE")
	v.writeRow(tn.Header)
	for _, row := range tn.Rows {
		v.writeRow(row)
	}
}
func (v *visitor) writeRow(row ast.TableRow) {
	v.WriteSep()
	v.WriteOpen()
	for i, cell := range row {
		if i > 0 {
			v.WriteSep()
		}
		v.writeCell(cell)
	}
	v.WriteClose()
}

var alignmentSymbol = map[ast.Alignment]string{
	ast.AlignDefault: "CELL",
	ast.AlignLeft:    "CELL-LEFT",
	ast.AlignCenter:  "CELL-CENTER",
	ast.AlignRight:   "CELL-RIGHT",
}

func (v *visitor) writeCell(cell *ast.TableCell) {
	v.WriteOpen()
	v.WriteSymbol(mapGet(alignmentSymbol, cell.Align))
	v.writeOptInlineSlice(cell.Inlines)
	v.WriteClose()
}

func (v *visitor) visitBLOB(bn *ast.BLOBNode) {
	v.WriteSymbol("BLOB")
	v.WriteSep()
	v.WriteEscapedSymbol(bn.Title)
	v.WriteSep()
	v.WriteEscapedSymbol(bn.Syntax)
	v.WriteSep()
	if bn.Syntax == api.ValueSyntaxSVG {
		v.WriteEscapedSymbol(string(bn.Blob))
	} else {
		v.WriteBase64Symbol(bn.Blob)
	}
}

func (v *visitor) visitEmbedBLOB(en *ast.EmbedBLOBNode) {
	v.WriteSymbol("EMBED-BLOB")
	v.writeAttributes(en.Attrs)
	v.WriteSep()
	v.WriteEscapedSymbol(en.Syntax)
	v.WriteSep()
	if en.Syntax == api.ValueSyntaxSVG {
		v.WriteEscapedSymbol(string(en.Blob))
	} else {
		v.WriteBase64Symbol(en.Blob)
	}
	v.writeOptInlineSlice(en.Inlines)
}

var mapFormatKind = map[ast.FormatKind]string{
	ast.FormatEmph:   "FORMAT-EMPH",
	ast.FormatStrong: "FORMAT-STRONG",
	ast.FormatDelete: "FORMAT-DELETE",
	ast.FormatInsert: "FORMAT-INSERT",
	ast.FormatSuper:  "FORMAT-SUPER",
	ast.FormatSub:    "FORMAT-SUB",
	ast.FormatQuote:  "FORMAT-QUOTE",
	ast.FormatSpan:   "FORMAT-SPAN",
}

var mapLiteralKind = map[ast.LiteralKind]string{
	ast.LiteralZettel:  "LITERAL-ZETTEL",
	ast.LiteralProg:    "LITERAL-CODE",
	ast.LiteralInput:   "LITERAL-INPUT",
	ast.LiteralOutput:  "LITERAL-OUTPUT",
	ast.LiteralComment: "LITERAL-COMMENT",
	ast.LiteralHTML:    "LITERAL-HTML",
	ast.LiteralMath:    "LITERAL-MATH",
}

func (v *visitor) writeBlockSlice(bs *ast.BlockSlice) {
	for i, n := range *bs {
		if i > 0 {
			v.WriteSep()
		}
		ast.Walk(v, n)
	}
}
func (v *visitor) writeInlineSlice(is ast.InlineSlice) {
	for i, n := range is {
		if i > 0 {
			v.WriteSep()
		}
		ast.Walk(v, n)
	}
}
func (v *visitor) writeOptInlineSlice(is ast.InlineSlice) {
	if len(is) > 0 {
		v.WriteSep()
		v.writeInlineSlice(is)
	}
}

func (v *visitor) writeAttributes(a zjson.Attributes) {
	v.WriteSep()
	if a.IsEmpty() {
		v.WriteSymbol("()")
		return
	}
	v.WriteOpen()
	for i, k := range a.Keys() {
		if i > 0 {
			v.WriteSep()
		}
		v.WriteStringSymbol(k)
		v.WriteSep()
		v.WriteStringSymbol(a[k])
	}
	v.WriteClose()
}

var mapRefState = map[ast.RefState]string{
	ast.RefStateInvalid:  "INVALID",
	ast.RefStateZettel:   "ZETTEL",
	ast.RefStateSelf:     "SELF",
	ast.RefStateFound:    "FOUND",
	ast.RefStateBroken:   "BROKEN",
	ast.RefStateHosted:   "HOSTED",
	ast.RefStateBased:    "BASED",
	ast.RefStateExternal: "EXTERNAL",
}

func (v *visitor) writeReference(ref *ast.Reference) {
	v.WriteSep()
	v.WriteOpen()
	v.WriteSymbol(mapGet(mapRefState, ref.State))
	v.WriteSep()
	v.WriteStringSymbol(ref.Value)
	v.WriteClose()
}

var mapMetaType = map[*meta.DescriptionType]string{
	meta.TypeCredential:   "CREDENTIAL",
	meta.TypeEmpty:        "EMPTY-STRING",
	meta.TypeID:           "ZID",
	meta.TypeIDSet:        "ZID-SET",
	meta.TypeNumber:       "NUMBER",
	meta.TypeString:       "STRING",
	meta.TypeTagSet:       "TAG-SET",
	meta.TypeTimestamp:    "TIMESTAMP",
	meta.TypeURL:          "URL",
	meta.TypeWord:         "WORD",
	meta.TypeWordSet:      "WORD-SET",
	meta.TypeZettelmarkup: "ZETTELMARKUP",
}

func (v *visitor) writeMeta(m *meta.Meta, evalMeta encoder.EvalMetaFunc) {
	v.WriteOpen()
	for i, p := range m.ComputedPairs() {
		if i > 0 {
			v.WriteSep()
		}
		key := p.Key
		t := m.Type(key)
		v.WriteOpen()
		v.WriteSymbol(mapGet(mapMetaType, t))
		v.WriteSep()
		v.WriteStringSymbol(key)
		if t.IsSet {
			for _, val := range meta.ListFromValue(p.Value) {
				v.WriteSep()
				v.WriteEscapedSymbol(val)
			}
		} else if t == meta.TypeZettelmarkup {
			v.WriteSep()
			is := evalMeta(p.Value)
			ast.Walk(v, &is)
		} else {
			v.WriteSep()
			v.WriteEscapedSymbol(p.Value)
		}
		v.WriteClose()
	}
	v.WriteClose()
}

func mapGet[T comparable](m map[T]string, k T) string {
	if result, found := m[k]; found {
		return result
	}
	log.Println("MISS", k, m)
	return fmt.Sprintf("**%v:not-found**", k)
}

var (
	bSpace = []byte{' '}
	bOpen  = []byte{'('}
	bClose = []byte{')'}
	bQuote = []byte{'"'}
)

func (v *visitor) WriteSep() {
	if v.err == nil {
		l, err := v.w.Write(bSpace)
		v.length += l
		v.err = err
	}
}
func (v *visitor) WriteOpen() {
	if v.err == nil {
		l, err := v.w.Write(bOpen)
		v.length += l
		v.err = err
	}
}
func (v *visitor) WriteClose() {
	if v.err == nil {
		l, err := v.w.Write(bClose)
		v.length += l
		v.err = err
	}
}
func (v *visitor) WriteSymbol(s string) {
	if v.err == nil {
		l, err := io.WriteString(v.w, s)
		v.length += l
		v.err = err
	}
}
func (v *visitor) WriteStringSymbol(s string) {
	if v.err == nil {
		l, err := v.w.Write(bQuote)
		v.length += l
		if err == nil {
			l, err = io.WriteString(v.w, s)
			v.length += l
			if err == nil {
				l, err = v.w.Write(bQuote)
				v.length += l
			}
		}
		v.err = err
	}
}
func (v *visitor) WriteEscapedSymbol(s string) {
	if v.err != nil {
		return
	}
	l, err := v.w.Write(bQuote)
	v.length += l
	if err == nil {
		l, err = strfun.JSONEscape(v.w, s)
		v.length += l
		if err == nil {
			l, err = v.w.Write(bQuote)
			v.length += l
		}
	}
	v.err = err
}
func (v *visitor) WriteBase64Symbol(p []byte) {
	if v.err != nil {
		return
	}
	l, err := v.w.Write(bQuote)
	v.length += l
	if err == nil {
		l, err = encodeBase64(v.w, p)
		v.length += l
		if err == nil {
			l, err = v.w.Write(bQuote)
			v.length += l
		}
	}
	v.err = err
}

func encodeBase64(w io.Writer, p []byte) (int, error) {
	encoder := base64.NewEncoder(base64.StdEncoding, w)
	l, err := encoder.Write(p)
	err1 := encoder.Close()
	if err == nil {
		err = err1
	}
	return l, err
}
