//-----------------------------------------------------------------------------
// Copyright (c) 2022-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2022-present Detlef Stern
//-----------------------------------------------------------------------------

package szenc

import (
	"encoding/base64"
	"fmt"
	"strings"

	"zettelstore.de/client.fossil/attrs"
	"zettelstore.de/client.fossil/sz"
	"zettelstore.de/sx.fossil"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/zettel/meta"
)

// NewTransformer returns a new transformer to create s-expressions from AST nodes.
func NewTransformer() *Transformer {
	t := Transformer{}

	return &t
}

type Transformer struct {
	inVerse bool
}

func (t *Transformer) GetSz(node ast.Node) *sx.Pair {
	switch n := node.(type) {
	case *ast.BlockSlice:
		return t.getBlockSlice(n)
	case *ast.InlineSlice:
		return t.getInlineSlice(*n)
	case *ast.ParaNode:
		return t.getInlineSlice(n.Inlines).Tail().Cons(sz.SymPara)
	case *ast.VerbatimNode:
		return sx.MakeList(
			mapGetS(mapVerbatimKindS, n.Kind),
			getAttributes(n.Attrs),
			sx.String(string(n.Content)),
		)
	case *ast.RegionNode:
		return t.getRegion(n)
	case *ast.HeadingNode:
		return sx.MakeList(
			sz.SymHeading,
			sx.Int64(int64(n.Level)),
			getAttributes(n.Attrs),
			sx.String(n.Slug),
			sx.String(n.Fragment),
			t.getInlineSlice(n.Inlines),
		)
	case *ast.HRuleNode:
		return sx.MakeList(sz.SymThematic, getAttributes(n.Attrs))
	case *ast.NestedListNode:
		return t.getNestedList(n)
	case *ast.DescriptionListNode:
		return t.getDescriptionList(n)
	case *ast.TableNode:
		return t.getTable(n)
	case *ast.TranscludeNode:
		return sx.MakeList(sz.SymTransclude, getAttributes(n.Attrs), getReference(n.Ref))
	case *ast.BLOBNode:
		return t.getBLOB(n)
	case *ast.TextNode:
		return sx.MakeList(sz.SymText, sx.String(n.Text))
	case *ast.SpaceNode:
		if t.inVerse {
			return sx.MakeList(sz.SymSpace, sx.String(n.Lexeme))
		}
		return sx.MakeList(sz.SymSpace)
	case *ast.BreakNode:
		if n.Hard {
			return sx.MakeList(sz.SymHard)
		}
		return sx.MakeList(sz.SymSoft)
	case *ast.LinkNode:
		return t.getLink(n)
	case *ast.EmbedRefNode:
		return t.getInlineSlice(n.Inlines).Tail().
			Cons(sx.String(n.Syntax)).
			Cons(getReference(n.Ref)).
			Cons(getAttributes(n.Attrs)).
			Cons(sz.SymEmbed)
	case *ast.EmbedBLOBNode:
		return t.getEmbedBLOB(n)
	case *ast.CiteNode:
		return t.getInlineSlice(n.Inlines).Tail().
			Cons(sx.String(n.Key)).
			Cons(getAttributes(n.Attrs)).
			Cons(sz.SymCite)
	case *ast.FootnoteNode:
		// (ENDNODE attrs (INLINE InlineElement ...))
		text := sx.Nil().Cons(t.getInlineSlice(n.Inlines))
		return text.Cons(getAttributes(n.Attrs)).Cons(sz.SymEndnote)
	case *ast.MarkNode:
		return t.getInlineSlice(n.Inlines).Tail().
			Cons(sx.String(n.Fragment)).
			Cons(sx.String(n.Slug)).
			Cons(sx.String(n.Mark)).
			Cons(sz.SymMark)
	case *ast.FormatNode:
		return t.getInlineSlice(n.Inlines).Tail().
			Cons(getAttributes(n.Attrs)).
			Cons(mapGetS(mapFormatKindS, n.Kind))
	case *ast.LiteralNode:
		return sx.MakeList(
			mapGetS(mapLiteralKindS, n.Kind),
			getAttributes(n.Attrs),
			sx.String(string(n.Content)),
		)
	}
	return sx.MakeList(sz.SymUnknown, sx.String(fmt.Sprintf("%T %v", node, node)))
}

var mapVerbatimKindS = map[ast.VerbatimKind]sx.Symbol{
	ast.VerbatimZettel:  sz.SymVerbatimZettel,
	ast.VerbatimProg:    sz.SymVerbatimProg,
	ast.VerbatimEval:    sz.SymVerbatimEval,
	ast.VerbatimMath:    sz.SymVerbatimMath,
	ast.VerbatimComment: sz.SymVerbatimComment,
	ast.VerbatimHTML:    sz.SymVerbatimHTML,
}

var mapFormatKindS = map[ast.FormatKind]sx.Symbol{
	ast.FormatEmph:   sz.SymFormatEmph,
	ast.FormatStrong: sz.SymFormatStrong,
	ast.FormatDelete: sz.SymFormatDelete,
	ast.FormatInsert: sz.SymFormatInsert,
	ast.FormatSuper:  sz.SymFormatSuper,
	ast.FormatSub:    sz.SymFormatSub,
	ast.FormatQuote:  sz.SymFormatQuote,
	ast.FormatMark:   sz.SymFormatMark,
	ast.FormatSpan:   sz.SymFormatSpan,
}

var mapLiteralKindS = map[ast.LiteralKind]sx.Symbol{
	ast.LiteralZettel:  sz.SymLiteralZettel,
	ast.LiteralProg:    sz.SymLiteralProg,
	ast.LiteralInput:   sz.SymLiteralInput,
	ast.LiteralOutput:  sz.SymLiteralOutput,
	ast.LiteralComment: sz.SymLiteralComment,
	ast.LiteralHTML:    sz.SymLiteralHTML,
	ast.LiteralMath:    sz.SymLiteralMath,
}

var mapRegionKindS = map[ast.RegionKind]sx.Symbol{
	ast.RegionSpan:  sz.SymRegionBlock,
	ast.RegionQuote: sz.SymRegionQuote,
	ast.RegionVerse: sz.SymRegionVerse,
}

func (t *Transformer) getRegion(rn *ast.RegionNode) *sx.Pair {
	saveInVerse := t.inVerse
	if rn.Kind == ast.RegionVerse {
		t.inVerse = true
	}
	symBlocks := t.GetSz(&rn.Blocks)
	t.inVerse = saveInVerse
	return sx.MakeList(
		mapGetS(mapRegionKindS, rn.Kind),
		getAttributes(rn.Attrs),
		symBlocks,
		t.GetSz(&rn.Inlines),
	)
}

var mapNestedListKindS = map[ast.NestedListKind]sx.Symbol{
	ast.NestedListOrdered:   sz.SymListOrdered,
	ast.NestedListUnordered: sz.SymListUnordered,
	ast.NestedListQuote:     sz.SymListQuote,
}

func (t *Transformer) getNestedList(ln *ast.NestedListNode) *sx.Pair {
	nlistObjs := make([]sx.Object, len(ln.Items)+1)
	nlistObjs[0] = mapGetS(mapNestedListKindS, ln.Kind)
	isCompact := isCompactList(ln.Items)
	for i, item := range ln.Items {
		if isCompact && len(item) > 0 {
			paragraph := t.GetSz(item[0])
			nlistObjs[i+1] = paragraph.Tail().Cons(sz.SymInline)
			continue
		}
		itemObjs := make([]sx.Object, len(item))
		for j, in := range item {
			itemObjs[j] = t.GetSz(in)
		}
		if isCompact {
			nlistObjs[i+1] = sx.MakeList(itemObjs...).Cons(sz.SymInline)
		} else {
			nlistObjs[i+1] = sx.MakeList(itemObjs...).Cons(sz.SymBlock)
		}
	}
	return sx.MakeList(nlistObjs...)
}
func isCompactList(itemSlice []ast.ItemSlice) bool {
	for _, items := range itemSlice {
		if len(items) > 1 {
			return false
		}
		if len(items) == 1 {
			if _, ok := items[0].(*ast.ParaNode); !ok {
				return false
			}
		}
	}
	return true
}

func (t *Transformer) getDescriptionList(dn *ast.DescriptionListNode) *sx.Pair {
	dlObjs := make([]sx.Object, 2*len(dn.Descriptions)+1)
	dlObjs[0] = sz.SymDescription
	for i, def := range dn.Descriptions {
		dlObjs[2*i+1] = t.getInlineSlice(def.Term)
		descObjs := make([]sx.Object, len(def.Descriptions))
		for j, b := range def.Descriptions {
			dVal := make([]sx.Object, len(b))
			for k, dn := range b {
				dVal[k] = t.GetSz(dn)
			}
			descObjs[j] = sx.MakeList(dVal...).Cons(sz.SymBlock)
		}
		dlObjs[2*i+2] = sx.MakeList(descObjs...).Cons(sz.SymBlock)
	}
	return sx.MakeList(dlObjs...)
}

func (t *Transformer) getTable(tn *ast.TableNode) *sx.Pair {
	tObjs := make([]sx.Object, len(tn.Rows)+2)
	tObjs[0] = sz.SymTable
	tObjs[1] = t.getHeader(tn.Header)
	for i, row := range tn.Rows {
		tObjs[i+2] = t.getRow(row)
	}
	return sx.MakeList(tObjs...)
}
func (t *Transformer) getHeader(header ast.TableRow) *sx.Pair {
	if len(header) == 0 {
		return nil
	}
	return t.getRow(header)
}
func (t *Transformer) getRow(row ast.TableRow) *sx.Pair {
	rObjs := make([]sx.Object, len(row))
	for i, cell := range row {
		rObjs[i] = t.getCell(cell)
	}
	return sx.MakeList(rObjs...)
}

var alignmentSymbolS = map[ast.Alignment]sx.Symbol{
	ast.AlignDefault: sz.SymCell,
	ast.AlignLeft:    sz.SymCellLeft,
	ast.AlignCenter:  sz.SymCellCenter,
	ast.AlignRight:   sz.SymCellRight,
}

func (t *Transformer) getCell(cell *ast.TableCell) *sx.Pair {
	return t.getInlineSlice(cell.Inlines).Tail().Cons(mapGetS(alignmentSymbolS, cell.Align))
}

func (t *Transformer) getBLOB(bn *ast.BLOBNode) *sx.Pair {
	var lastObj sx.Object
	if bn.Syntax == meta.SyntaxSVG {
		lastObj = sx.String(string(bn.Blob))
	} else {
		lastObj = getBase64String(bn.Blob)
	}
	return sx.MakeList(
		sz.SymBLOB,
		t.getInlineSlice(bn.Description),
		sx.String(bn.Syntax),
		lastObj,
	)
}

var mapRefStateLink = map[ast.RefState]sx.Symbol{
	ast.RefStateInvalid:  sz.SymLinkInvalid,
	ast.RefStateZettel:   sz.SymLinkZettel,
	ast.RefStateSelf:     sz.SymLinkSelf,
	ast.RefStateFound:    sz.SymLinkFound,
	ast.RefStateBroken:   sz.SymLinkBroken,
	ast.RefStateHosted:   sz.SymLinkHosted,
	ast.RefStateBased:    sz.SymLinkBased,
	ast.RefStateQuery:    sz.SymLinkQuery,
	ast.RefStateExternal: sz.SymLinkExternal,
}

func (t *Transformer) getLink(ln *ast.LinkNode) *sx.Pair {
	return t.getInlineSlice(ln.Inlines).Tail().
		Cons(sx.String(ln.Ref.Value)).
		Cons(getAttributes(ln.Attrs)).
		Cons(mapGetS(mapRefStateLink, ln.Ref.State))
}

func (t *Transformer) getEmbedBLOB(en *ast.EmbedBLOBNode) *sx.Pair {
	tail := t.getInlineSlice(en.Inlines).Tail()
	if en.Syntax == meta.SyntaxSVG {
		tail = tail.Cons(sx.String(string(en.Blob)))
	} else {
		tail = tail.Cons(getBase64String(en.Blob))
	}
	return tail.Cons(sx.String(en.Syntax)).Cons(getAttributes(en.Attrs)).Cons(sz.SymEmbedBLOB)
}

func (t *Transformer) getBlockSlice(bs *ast.BlockSlice) *sx.Pair {
	objs := make([]sx.Object, len(*bs))
	for i, n := range *bs {
		objs[i] = t.GetSz(n)
	}
	return sx.MakeList(objs...).Cons(sz.SymBlock)
}
func (t *Transformer) getInlineSlice(is ast.InlineSlice) *sx.Pair {
	objs := make([]sx.Object, len(is))
	for i, n := range is {
		objs[i] = t.GetSz(n)
	}
	return sx.MakeList(objs...).Cons(sz.SymInline)
}

func getAttributes(a attrs.Attributes) sx.Object {
	if a.IsEmpty() {
		return sx.Nil()
	}
	keys := a.Keys()
	objs := make([]sx.Object, 0, len(keys))
	for _, k := range keys {
		objs = append(objs, sx.Cons(sx.String(k), sx.String(a[k])))
	}
	return sx.MakeList(objs...)
}

var mapRefStateS = map[ast.RefState]sx.Symbol{
	ast.RefStateInvalid:  sz.SymRefStateInvalid,
	ast.RefStateZettel:   sz.SymRefStateZettel,
	ast.RefStateSelf:     sz.SymRefStateSelf,
	ast.RefStateFound:    sz.SymRefStateFound,
	ast.RefStateBroken:   sz.SymRefStateBroken,
	ast.RefStateHosted:   sz.SymRefStateHosted,
	ast.RefStateBased:    sz.SymRefStateBased,
	ast.RefStateQuery:    sz.SymRefStateQuery,
	ast.RefStateExternal: sz.SymRefStateExternal,
}

func getReference(ref *ast.Reference) *sx.Pair {
	return sx.MakeList(mapGetS(mapRefStateS, ref.State), sx.String(ref.Value))
}

var mapMetaTypeS = map[*meta.DescriptionType]sx.Symbol{
	meta.TypeCredential:   sz.SymTypeCredential,
	meta.TypeEmpty:        sz.SymTypeEmpty,
	meta.TypeID:           sz.SymTypeID,
	meta.TypeIDSet:        sz.SymTypeIDSet,
	meta.TypeNumber:       sz.SymTypeNumber,
	meta.TypeString:       sz.SymTypeString,
	meta.TypeTagSet:       sz.SymTypeTagSet,
	meta.TypeTimestamp:    sz.SymTypeTimestamp,
	meta.TypeURL:          sz.SymTypeURL,
	meta.TypeWord:         sz.SymTypeWord,
	meta.TypeWordSet:      sz.SymTypeWordSet,
	meta.TypeZettelmarkup: sz.SymTypeZettelmarkup,
}

func (t *Transformer) GetMeta(m *meta.Meta, evalMeta encoder.EvalMetaFunc) *sx.Pair {
	pairs := m.ComputedPairs()
	objs := make([]sx.Object, 0, len(pairs))
	for _, p := range pairs {
		key := p.Key
		ty := m.Type(key)
		symType := mapGetS(mapMetaTypeS, ty)
		var obj sx.Object
		if ty.IsSet {
			setList := meta.ListFromValue(p.Value)
			setObjs := make([]sx.Object, len(setList))
			for i, val := range setList {
				setObjs[i] = sx.String(val)
			}
			obj = sx.MakeList(setObjs...)
		} else if ty == meta.TypeZettelmarkup {
			is := evalMeta(p.Value)
			obj = t.GetSz(&is)
		} else {
			obj = sx.String(p.Value)
		}
		objs = append(objs, sx.Nil().Cons(obj).Cons(sx.Symbol(key)).Cons(symType))
	}
	return sx.MakeList(objs...).Cons(sz.SymMeta)
}

func mapGetS[T comparable](m map[T]sx.Symbol, k T) sx.Symbol {
	if result, found := m[k]; found {
		return result
	}
	return sx.Symbol(fmt.Sprintf("**%v:NOT-FOUND**", k))
}

func getBase64String(data []byte) sx.String {
	var sb strings.Builder
	encoder := base64.NewEncoder(base64.StdEncoding, &sb)
	_, err := encoder.Write(data)
	if err == nil {
		err = encoder.Close()
	}
	if err == nil {
		return sx.String(sb.String())
	}
	return sx.String("")
}
