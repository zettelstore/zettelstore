//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package textenc encodes the abstract syntax tree into its text.
package textenc

import (
	"io"

	"zettelstore.de/z/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/parser"
)

func init() {
	encoder.Register(api.EncoderText, encoder.Info{
		Create: func(*encoder.Environment) encoder.Encoder { return &textEncoder{} },
	})
}

type textEncoder struct{}

// WriteZettel writes metadata and content.
func (te *textEncoder) WriteZettel(w io.Writer, zn *ast.ZettelNode) (int, error) {
	v := newVisitor(w)
	te.WriteMeta(&v.b, zn.InhMeta)
	v.acceptBlockList(zn.Ast)
	length, err := v.b.Flush()
	return length, err
}

// WriteMeta encodes metadata as text.
func (te *textEncoder) WriteMeta(w io.Writer, m *meta.Meta) (int, error) {
	buf := encoder.NewBufWriter(w)
	for _, pair := range m.Pairs(true) {
		switch meta.Type(pair.Key) {
		case meta.TypeBool:
			writeBool(&buf, meta.BoolValue(pair.Value))
		case meta.TypeTagSet:
			writeTagSet(&buf, meta.ListFromValue(pair.Value))
		case meta.TypeZettelmarkup:
			te.writeZettelmarkup(&buf, pair.Value)
		default:
			buf.WriteString(pair.Value)
		}
		buf.WriteByte('\n')
	}
	length, err := buf.Flush()
	return length, err
}

func writeBool(buf *encoder.BufWriter, b bool) {
	if b {
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

func (te *textEncoder) writeZettelmarkup(buf *encoder.BufWriter, zmk string) {
	if iln := parser.ParseMetadata(zmk); iln != nil {
		te.WriteInlines(buf, iln)
	}
}
func (te *textEncoder) WriteContent(w io.Writer, zn *ast.ZettelNode) (int, error) {
	return te.WriteBlocks(w, zn.Ast)
}

// WriteBlocks writes the content of a block slice to the writer.
func (te *textEncoder) WriteBlocks(w io.Writer, bln *ast.BlockListNode) (int, error) {
	v := newVisitor(w)
	v.acceptBlockList(bln)
	length, err := v.b.Flush()
	return length, err
}

// WriteInlines writes an inline slice to the writer
func (te *textEncoder) WriteInlines(w io.Writer, iln *ast.InlineListNode) (int, error) {
	v := newVisitor(w)
	ast.Walk(v, iln)
	length, err := v.b.Flush()
	return length, err
}

// visitor writes the abstract syntax tree to an io.Writer.
type visitor struct {
	b encoder.BufWriter
}

func newVisitor(w io.Writer) *visitor {
	return &visitor{b: encoder.NewBufWriter(w)}
}

func (v *visitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.BlockListNode:
		v.acceptBlockList(n)
	case *ast.VerbatimNode:
		if n.Kind == ast.VerbatimComment {
			return nil
		}
		for i, line := range n.Lines {
			v.writePosChar(i, '\n')
			v.b.WriteString(line)
		}
		return nil
	case *ast.RegionNode:
		v.acceptBlockList(n.Blocks)
		if n.Inlines != nil {
			v.b.WriteByte('\n')
			ast.Walk(v, n.Inlines)
		}
		return nil
	case *ast.NestedListNode:
		for i, item := range n.Items {
			v.writePosChar(i, '\n')
			for j, it := range item {
				v.writePosChar(j, '\n')
				ast.Walk(v, it)
			}
		}
		return nil
	case *ast.DescriptionListNode:
		for i, descr := range n.Descriptions {
			v.writePosChar(i, '\n')
			ast.Walk(v, descr.Term)
			for _, b := range descr.Descriptions {
				v.b.WriteByte('\n')
				for k, d := range b {
					v.writePosChar(k, '\n')
					ast.Walk(v, d)
				}
			}
		}
		return nil
	case *ast.TableNode:
		if len(n.Header) > 0 {
			v.writeRow(n.Header)
			v.b.WriteByte('\n')
		}
		for i, row := range n.Rows {
			v.writePosChar(i, '\n')
			v.writeRow(row)
		}
		return nil
	case *ast.TextNode:
		v.b.WriteString(n.Text)
		return nil
	case *ast.TagNode:
		v.b.WriteStrings("#", n.Tag)
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
			ast.Walk(v, n.Inlines)
		}
		return nil
	case *ast.FootnoteNode:
		v.b.WriteByte(' ')
		return v // No 'return nil' to write text
	case *ast.LiteralNode:
		if n.Kind != ast.LiteralComment {
			v.b.WriteString(n.Text)
		}
	}
	return v
}

func (v *visitor) writeRow(row ast.TableRow) {
	for i, cell := range row {
		v.writePosChar(i, ' ')
		ast.Walk(v, cell.Inlines)
	}
}

func (v *visitor) acceptBlockList(bns *ast.BlockListNode) {
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
