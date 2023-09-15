//-----------------------------------------------------------------------------
// Copyright (c) 2022-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
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
	sf := sx.MakeMappedFactory(1024)
	t := Transformer{sf: sf}
	t.zetSyms.InitializeZettelSymbols(sf)

	t.mapVerbatimKindS = map[ast.VerbatimKind]*sx.Symbol{
		ast.VerbatimZettel:  t.zetSyms.SymVerbatimZettel,
		ast.VerbatimProg:    t.zetSyms.SymVerbatimProg,
		ast.VerbatimEval:    t.zetSyms.SymVerbatimEval,
		ast.VerbatimMath:    t.zetSyms.SymVerbatimMath,
		ast.VerbatimComment: t.zetSyms.SymVerbatimComment,
		ast.VerbatimHTML:    t.zetSyms.SymVerbatimHTML,
	}

	t.mapRegionKindS = map[ast.RegionKind]*sx.Symbol{
		ast.RegionSpan:  t.zetSyms.SymRegionBlock,
		ast.RegionQuote: t.zetSyms.SymRegionQuote,
		ast.RegionVerse: t.zetSyms.SymRegionVerse,
	}
	t.mapNestedListKindS = map[ast.NestedListKind]*sx.Symbol{
		ast.NestedListOrdered:   t.zetSyms.SymListOrdered,
		ast.NestedListUnordered: t.zetSyms.SymListUnordered,
		ast.NestedListQuote:     t.zetSyms.SymListQuote,
	}
	t.alignmentSymbolS = map[ast.Alignment]*sx.Symbol{
		ast.AlignDefault: t.zetSyms.SymCell,
		ast.AlignLeft:    t.zetSyms.SymCellLeft,
		ast.AlignCenter:  t.zetSyms.SymCellCenter,
		ast.AlignRight:   t.zetSyms.SymCellRight,
	}
	t.mapRefStateLink = map[ast.RefState]*sx.Symbol{
		ast.RefStateInvalid:  t.zetSyms.SymLinkInvalid,
		ast.RefStateZettel:   t.zetSyms.SymLinkZettel,
		ast.RefStateSelf:     t.zetSyms.SymLinkSelf,
		ast.RefStateFound:    t.zetSyms.SymLinkFound,
		ast.RefStateBroken:   t.zetSyms.SymLinkBroken,
		ast.RefStateHosted:   t.zetSyms.SymLinkHosted,
		ast.RefStateBased:    t.zetSyms.SymLinkBased,
		ast.RefStateQuery:    t.zetSyms.SymLinkQuery,
		ast.RefStateExternal: t.zetSyms.SymLinkExternal,
	}
	t.mapFormatKindS = map[ast.FormatKind]*sx.Symbol{
		ast.FormatEmph:   t.zetSyms.SymFormatEmph,
		ast.FormatStrong: t.zetSyms.SymFormatStrong,
		ast.FormatDelete: t.zetSyms.SymFormatDelete,
		ast.FormatInsert: t.zetSyms.SymFormatInsert,
		ast.FormatSuper:  t.zetSyms.SymFormatSuper,
		ast.FormatSub:    t.zetSyms.SymFormatSub,
		ast.FormatQuote:  t.zetSyms.SymFormatQuote,
		ast.FormatSpan:   t.zetSyms.SymFormatSpan,
	}
	t.mapLiteralKindS = map[ast.LiteralKind]*sx.Symbol{
		ast.LiteralZettel:  t.zetSyms.SymLiteralZettel,
		ast.LiteralProg:    t.zetSyms.SymLiteralProg,
		ast.LiteralInput:   t.zetSyms.SymLiteralInput,
		ast.LiteralOutput:  t.zetSyms.SymLiteralOutput,
		ast.LiteralComment: t.zetSyms.SymLiteralComment,
		ast.LiteralHTML:    t.zetSyms.SymLiteralHTML,
		ast.LiteralMath:    t.zetSyms.SymLiteralMath,
	}
	t.mapRefStateS = map[ast.RefState]*sx.Symbol{
		ast.RefStateInvalid:  t.zetSyms.SymRefStateInvalid,
		ast.RefStateZettel:   t.zetSyms.SymRefStateZettel,
		ast.RefStateSelf:     t.zetSyms.SymRefStateSelf,
		ast.RefStateFound:    t.zetSyms.SymRefStateFound,
		ast.RefStateBroken:   t.zetSyms.SymRefStateBroken,
		ast.RefStateHosted:   t.zetSyms.SymRefStateHosted,
		ast.RefStateBased:    t.zetSyms.SymRefStateBased,
		ast.RefStateQuery:    t.zetSyms.SymRefStateQuery,
		ast.RefStateExternal: t.zetSyms.SymRefStateExternal,
	}
	t.mapMetaTypeS = map[*meta.DescriptionType]*sx.Symbol{
		meta.TypeCredential:   t.zetSyms.SymTypeCredential,
		meta.TypeEmpty:        t.zetSyms.SymTypeEmpty,
		meta.TypeID:           t.zetSyms.SymTypeID,
		meta.TypeIDSet:        t.zetSyms.SymTypeIDSet,
		meta.TypeNumber:       t.zetSyms.SymTypeNumber,
		meta.TypeString:       t.zetSyms.SymTypeString,
		meta.TypeTagSet:       t.zetSyms.SymTypeTagSet,
		meta.TypeTimestamp:    t.zetSyms.SymTypeTimestamp,
		meta.TypeURL:          t.zetSyms.SymTypeURL,
		meta.TypeWord:         t.zetSyms.SymTypeWord,
		meta.TypeWordSet:      t.zetSyms.SymTypeWordSet,
		meta.TypeZettelmarkup: t.zetSyms.SymTypeZettelmarkup,
	}
	return &t
}

type Transformer struct {
	sf                 sx.SymbolFactory
	zetSyms            sz.ZettelSymbols
	mapVerbatimKindS   map[ast.VerbatimKind]*sx.Symbol
	mapRegionKindS     map[ast.RegionKind]*sx.Symbol
	mapNestedListKindS map[ast.NestedListKind]*sx.Symbol
	alignmentSymbolS   map[ast.Alignment]*sx.Symbol
	mapRefStateLink    map[ast.RefState]*sx.Symbol
	mapFormatKindS     map[ast.FormatKind]*sx.Symbol
	mapLiteralKindS    map[ast.LiteralKind]*sx.Symbol
	mapRefStateS       map[ast.RefState]*sx.Symbol
	mapMetaTypeS       map[*meta.DescriptionType]*sx.Symbol
	inVerse            bool
}

func (t *Transformer) GetSz(node ast.Node) *sx.Pair {
	switch n := node.(type) {
	case *ast.BlockSlice:
		return t.getBlockSlice(n)
	case *ast.InlineSlice:
		return t.getInlineSlice(*n)
	case *ast.ParaNode:
		return t.getInlineSlice(n.Inlines).Tail().Cons(t.zetSyms.SymPara)
	case *ast.VerbatimNode:
		return sx.MakeList(
			mapGetS(t, t.mapVerbatimKindS, n.Kind),
			t.getAttributes(n.Attrs),
			sx.String(string(n.Content)),
		)
	case *ast.RegionNode:
		return t.getRegion(n)
	case *ast.HeadingNode:
		return sx.MakeList(
			t.zetSyms.SymHeading,
			sx.Int64(int64(n.Level)),
			t.getAttributes(n.Attrs),
			sx.String(n.Slug),
			sx.String(n.Fragment),
			t.getInlineSlice(n.Inlines),
		)
	case *ast.HRuleNode:
		return sx.MakeList(t.zetSyms.SymThematic, t.getAttributes(n.Attrs))
	case *ast.NestedListNode:
		return t.getNestedList(n)
	case *ast.DescriptionListNode:
		return t.getDescriptionList(n)
	case *ast.TableNode:
		return t.getTable(n)
	case *ast.TranscludeNode:
		return sx.MakeList(t.zetSyms.SymTransclude, t.getAttributes(n.Attrs), t.getReference(n.Ref))
	case *ast.BLOBNode:
		return t.getBLOB(n)
	case *ast.TextNode:
		return sx.MakeList(t.zetSyms.SymText, sx.String(n.Text))
	case *ast.SpaceNode:
		if t.inVerse {
			return sx.MakeList(t.zetSyms.SymSpace, sx.String(n.Lexeme))
		}
		return sx.MakeList(t.zetSyms.SymSpace)
	case *ast.BreakNode:
		if n.Hard {
			return sx.MakeList(t.zetSyms.SymHard)
		}
		return sx.MakeList(t.zetSyms.SymSoft)
	case *ast.LinkNode:
		return t.getLink(n)
	case *ast.EmbedRefNode:
		return t.getInlineSlice(n.Inlines).Tail().
			Cons(sx.String(n.Syntax)).
			Cons(t.getReference(n.Ref)).
			Cons(t.getAttributes(n.Attrs)).
			Cons(t.zetSyms.SymEmbed)
	case *ast.EmbedBLOBNode:
		return t.getEmbedBLOB(n)
	case *ast.CiteNode:
		return t.getInlineSlice(n.Inlines).Tail().
			Cons(sx.String(n.Key)).
			Cons(t.getAttributes(n.Attrs)).
			Cons(t.zetSyms.SymCite)
	case *ast.FootnoteNode:
		text := sx.Nil().Cons(sx.Nil().Cons(t.getInlineSlice(n.Inlines)).Cons(t.zetSyms.SymQuote))
		return text.Cons(t.getAttributes(n.Attrs)).Cons(t.zetSyms.SymEndnote)
	case *ast.MarkNode:
		return t.getInlineSlice(n.Inlines).Tail().
			Cons(sx.String(n.Fragment)).
			Cons(sx.String(n.Slug)).
			Cons(sx.String(n.Mark)).
			Cons(t.zetSyms.SymMark)
	case *ast.FormatNode:
		return t.getInlineSlice(n.Inlines).Tail().
			Cons(t.getAttributes(n.Attrs)).
			Cons(mapGetS(t, t.mapFormatKindS, n.Kind))
	case *ast.LiteralNode:
		return sx.MakeList(
			mapGetS(t, t.mapLiteralKindS, n.Kind),
			t.getAttributes(n.Attrs),
			sx.String(string(n.Content)),
		)
	}
	return sx.MakeList(t.zetSyms.SymUnknown, sx.String(fmt.Sprintf("%T %v", node, node)))
}

func (t *Transformer) getRegion(rn *ast.RegionNode) *sx.Pair {
	saveInVerse := t.inVerse
	if rn.Kind == ast.RegionVerse {
		t.inVerse = true
	}
	symBlocks := t.GetSz(&rn.Blocks)
	t.inVerse = saveInVerse
	return sx.MakeList(
		mapGetS(t, t.mapRegionKindS, rn.Kind),
		t.getAttributes(rn.Attrs),
		symBlocks,
		t.GetSz(&rn.Inlines),
	)
}

func (t *Transformer) getNestedList(ln *ast.NestedListNode) *sx.Pair {
	nlistObjs := make([]sx.Object, len(ln.Items)+1)
	nlistObjs[0] = mapGetS(t, t.mapNestedListKindS, ln.Kind)
	isCompact := isCompactList(ln.Items)
	for i, item := range ln.Items {
		if isCompact && len(item) > 0 {
			paragraph := t.GetSz(item[0])
			nlistObjs[i+1] = paragraph.Tail().Cons(t.zetSyms.SymInline)
			continue
		}
		itemObjs := make([]sx.Object, len(item))
		for j, in := range item {
			itemObjs[j] = t.GetSz(in)
		}
		if isCompact {
			nlistObjs[i+1] = sx.MakeList(itemObjs...).Cons(t.zetSyms.SymInline)
		} else {
			nlistObjs[i+1] = sx.MakeList(itemObjs...).Cons(t.zetSyms.SymBlock)
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
	dlObjs[0] = t.zetSyms.SymDescription
	for i, def := range dn.Descriptions {
		dlObjs[2*i+1] = t.getInlineSlice(def.Term)
		descObjs := make([]sx.Object, len(def.Descriptions))
		for j, b := range def.Descriptions {
			dVal := make([]sx.Object, len(b))
			for k, dn := range b {
				dVal[k] = t.GetSz(dn)
			}
			descObjs[j] = sx.MakeList(dVal...).Cons(t.zetSyms.SymBlock)
		}
		dlObjs[2*i+2] = sx.MakeList(descObjs...).Cons(t.zetSyms.SymBlock)
	}
	return sx.MakeList(dlObjs...)
}

func (t *Transformer) getTable(tn *ast.TableNode) *sx.Pair {
	tObjs := make([]sx.Object, len(tn.Rows)+2)
	tObjs[0] = t.zetSyms.SymTable
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
	return sx.MakeList(rObjs...).Cons(t.zetSyms.SymList)
}

func (t *Transformer) getCell(cell *ast.TableCell) *sx.Pair {
	return t.getInlineSlice(cell.Inlines).Tail().Cons(mapGetS(t, t.alignmentSymbolS, cell.Align))
}

func (t *Transformer) getBLOB(bn *ast.BLOBNode) *sx.Pair {
	var lastObj sx.Object
	if bn.Syntax == meta.SyntaxSVG {
		lastObj = sx.String(string(bn.Blob))
	} else {
		lastObj = getBase64String(bn.Blob)
	}
	return sx.MakeList(
		t.zetSyms.SymBLOB,
		t.getInlineSlice(bn.Description),
		sx.String(bn.Syntax),
		lastObj,
	)
}

func (t *Transformer) getLink(ln *ast.LinkNode) *sx.Pair {
	return t.getInlineSlice(ln.Inlines).Tail().
		Cons(sx.String(ln.Ref.Value)).
		Cons(t.getAttributes(ln.Attrs)).
		Cons(mapGetS(t, t.mapRefStateLink, ln.Ref.State))
}

func (t *Transformer) getEmbedBLOB(en *ast.EmbedBLOBNode) *sx.Pair {
	tail := t.getInlineSlice(en.Inlines).Tail()
	if en.Syntax == meta.SyntaxSVG {
		tail = tail.Cons(sx.String(string(en.Blob)))
	} else {
		tail = tail.Cons(getBase64String(en.Blob))
	}
	return tail.Cons(sx.String(en.Syntax)).Cons(t.getAttributes(en.Attrs)).Cons(t.zetSyms.SymEmbedBLOB)
}

func (t *Transformer) getBlockSlice(bs *ast.BlockSlice) *sx.Pair {
	objs := make([]sx.Object, len(*bs))
	for i, n := range *bs {
		objs[i] = t.GetSz(n)
	}
	return sx.MakeList(objs...).Cons(t.zetSyms.SymBlock)
}
func (t *Transformer) getInlineSlice(is ast.InlineSlice) *sx.Pair {
	objs := make([]sx.Object, len(is))
	for i, n := range is {
		objs[i] = t.GetSz(n)
	}
	return sx.MakeList(objs...).Cons(t.zetSyms.SymInline)
}

func (t *Transformer) getAttributes(a attrs.Attributes) sx.Object {
	if a.IsEmpty() {
		return sx.Nil()
	}
	keys := a.Keys()
	objs := make([]sx.Object, 0, len(keys))
	for _, k := range keys {
		objs = append(objs, sx.Cons(sx.String(k), sx.String(a[k])))
	}
	return sx.Nil().Cons(sx.MakeList(objs...)).Cons(t.zetSyms.SymQuote)
}

func (t *Transformer) getReference(ref *ast.Reference) *sx.Pair {
	return sx.MakeList(
		t.zetSyms.SymQuote,
		sx.MakeList(
			mapGetS(t, t.mapRefStateS, ref.State),
			sx.String(ref.Value),
		),
	)
}

func (t *Transformer) GetMeta(m *meta.Meta, evalMeta encoder.EvalMetaFunc) *sx.Pair {
	pairs := m.ComputedPairs()
	objs := make([]sx.Object, 0, len(pairs))
	for _, p := range pairs {
		key := p.Key
		ty := m.Type(key)
		symType := mapGetS(t, t.mapMetaTypeS, ty)
		var obj sx.Object
		if ty.IsSet {
			setList := meta.ListFromValue(p.Value)
			setObjs := make([]sx.Object, len(setList))
			for i, val := range setList {
				setObjs[i] = sx.String(val)
			}
			obj = sx.MakeList(setObjs...).Cons(t.zetSyms.SymList)
		} else if ty == meta.TypeZettelmarkup {
			is := evalMeta(p.Value)
			obj = t.GetSz(&is)
		} else {
			obj = sx.String(p.Value)
		}
		symKey := sx.MakeList(t.zetSyms.SymQuote, t.sf.MustMake(key))
		objs = append(objs, sx.Nil().Cons(obj).Cons(symKey).Cons(symType))
	}
	return sx.MakeList(objs...).Cons(t.zetSyms.SymMeta)
}

func mapGetS[T comparable](t *Transformer, m map[T]*sx.Symbol, k T) *sx.Symbol {
	if result, found := m[k]; found {
		return result
	}
	return t.sf.MustMake(fmt.Sprintf("**%v:NOT-FOUND**", k))
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
