//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package textenc encodes the abstract syntax tree into its text.
package textenc

import (
	"io"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
)

func init() {
	encoder.Register(api.EncoderText, encoder.Info{
		Create: func(*encoder.Environment) encoder.Encoder { return &textEncoder{} },
	})
}

type textEncoder struct{}

// WriteZettel writes metadata and content.
func (te *textEncoder) WriteZettel(w io.Writer, zn *ast.ZettelNode, evalMeta encoder.EvalMetaFunc) (int, error) {
	v := newVisitor(w)
	te.WriteMeta(&v.b, zn.InhMeta, evalMeta)
	v.visitBlockList(zn.Ast)
	length, err := v.b.Flush()
	return length, err
}

// WriteMeta encodes metadata as text.
func (te *textEncoder) WriteMeta(w io.Writer, m *meta.Meta, evalMeta encoder.EvalMetaFunc) (int, error) {
	buf := encoder.NewBufWriter(w)
	for _, pair := range m.ComputedPairs() {
		switch meta.Type(pair.Key) {
		case meta.TypeBool:
			writeBool(&buf, pair.Value)
		case meta.TypeTagSet:
			writeTagSet(&buf, meta.ListFromValue(pair.Value))
		case meta.TypeZettelmarkup:
			iln := evalMeta(pair.Value)
			te.WriteInlines(&buf, &iln)
		default:
			buf.WriteString(pair.Value)
		}
		buf.WriteByte('\n')
	}
	length, err := buf.Flush()
	return length, err
}

func writeBool(buf *encoder.BufWriter, val string) {
	if meta.BoolValue(val) {
		buf.WriteString("true")
	} else {
		buf.WriteString("false")
	}
}

func writeTagSet(buf *encoder.BufWriter, tags []string) {
	for i, tag := range tags {
		if i > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(meta.CleanTag(tag))
	}

}

func (te *textEncoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return te.WriteBlocks(w, zn.Ast)
}

// WriteBlocks writes the content of a block slice to the writer.
func (*textEncoder) WriteBlocks(w io.Writer, bln *ast.BlockListNode) (int, error) {
	v := newVisitor(w)
	v.visitBlockList(bln)
	length, err := v.b.Flush()
	return length, err
}

// WriteInlines writes an inline slice to the writer
func (*textEncoder) WriteInlines(w io.Writer, iln *ast.InlineListNode) (int, error) {
	v := newVisitor(w)
	ast.Walk(v, iln)
	length, err := v.b.Flush()
	return length, err
}

// visitor writes the abstract syntax tree to an io.Writer.
type visitor struct {
	b         encoder.BufWriter
	inlinePos int
}

func newVisitor(w io.Writer) *visitor {
	return &visitor{b: encoder.NewBufWriter(w)}
}

func (v *visitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.BlockListNode:
		v.visitBlockList(n)
	case *ast.InlineListNode:
		for i, in := range n.List {
			v.inlinePos = i
			ast.Walk(v, in)
		}
		v.inlinePos = 0
		return nil
	case *ast.VerbatimNode:
		v.visitVerbatim(n)
		return nil
	case *ast.RegionNode:
		v.visitBlockList(n.Blocks)
		if !n.Inlines.IsEmpty() {
			v.b.WriteByte('\n')
			ast.Walk(v, &n.Inlines)
		}
		return nil
	case *ast.NestedListNode:
		v.visitNestedList(n)
		return nil
	case *ast.DescriptionListNode:
		v.visitDescriptionList(n)
		return nil
	case *ast.TableNode:
		v.visitTable(n)
		return nil
	case *ast.TranscludeNode, *ast.BLOBNode:
		return nil
	case *ast.TextNode:
		v.b.WriteString(n.Text)
		return nil
	case *ast.TagNode:
		v.b.WriteString(n.Tag)
		return nil
	case *ast.SpaceNode:
		v.b.WriteByte(' ')
		return nil
	case *ast.BreakNode:
		if n.Hard {
			v.b.WriteByte('\n')
		} else {
			v.b.WriteByte(' ')
		}
		return nil
	case *ast.LinkNode:
		if !n.OnlyRef {
			ast.Walk(v, &n.Inlines)
		}
		return nil
	case *ast.FootnoteNode:
		if v.inlinePos > 0 {
			v.b.WriteByte(' ')
		}
		// No 'return nil' to write text
	case *ast.LiteralNode:
		if n.Kind != ast.LiteralComment {
			v.b.Write(n.Content)
		}
	}
	return v
}

func (v *visitor) visitVerbatim(vn *ast.VerbatimNode) {
	if vn.Kind == ast.VerbatimComment {
		return
	}
	v.b.Write(vn.Content)
}

func (v *visitor) visitNestedList(ln *ast.NestedListNode) {
	for i, item := range ln.Items {
		v.writePosChar(i, '\n')
		for j, it := range item {
			v.writePosChar(j, '\n')
			ast.Walk(v, it)
		}
	}
}

func (v *visitor) visitDescriptionList(dl *ast.DescriptionListNode) {
	for i, descr := range dl.Descriptions {
		v.writePosChar(i, '\n')
		ast.Walk(v, &descr.Term)
		for _, b := range descr.Descriptions {
			v.b.WriteByte('\n')
			for k, d := range b {
				v.writePosChar(k, '\n')
				ast.Walk(v, d)
			}
		}
	}
}

func (v *visitor) visitTable(tn *ast.TableNode) {
	if len(tn.Header) > 0 {
		v.writeRow(tn.Header)
		v.b.WriteByte('\n')
	}
	for i, row := range tn.Rows {
		v.writePosChar(i, '\n')
		v.writeRow(row)
	}
}

func (v *visitor) writeRow(row ast.TableRow) {
	for i, cell := range row {
		v.writePosChar(i, ' ')
		ast.Walk(v, &cell.Inlines)
	}
}

func (v *visitor) visitBlockList(bns *ast.BlockListNode) {
	for i, bn := range bns.List {
		v.writePosChar(i, '\n')
		ast.Walk(v, bn)
	}
}

func (v *visitor) writePosChar(pos int, ch byte) {
	if pos > 0 {
		v.b.WriteByte(ch)
	}
}
