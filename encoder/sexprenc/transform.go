//-----------------------------------------------------------------------------
// Copyright (c) 2022-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package sexprenc

import (
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"codeberg.org/t73fde/sxpf"
	"zettelstore.de/c/attrs"
	"zettelstore.de/c/sexpr"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
)

// NewTransformer returns a new transformer to create s-expressions from AST nodes.
func NewTransformer() *Transformer {
	sf := sxpf.MakeMappedFactory()
	t := Transformer{sf: sf}
	t.zetSyms.InitializeZettelSymbols(sf)

	t.mapVerbatimKindS = map[ast.VerbatimKind]*sxpf.Symbol{
		ast.VerbatimZettel:  t.zetSyms.SymVerbatimZettel,
		ast.VerbatimProg:    t.zetSyms.SymVerbatimProg,
		ast.VerbatimEval:    t.zetSyms.SymVerbatimEval,
		ast.VerbatimMath:    t.zetSyms.SymVerbatimMath,
		ast.VerbatimComment: t.zetSyms.SymVerbatimComment,
		ast.VerbatimHTML:    t.zetSyms.SymVerbatimHTML,
	}

	t.mapRegionKindS = map[ast.RegionKind]*sxpf.Symbol{
		ast.RegionSpan:  t.zetSyms.SymRegionBlock,
		ast.RegionQuote: t.zetSyms.SymRegionQuote,
		ast.RegionVerse: t.zetSyms.SymRegionVerse,
	}
	t.mapNestedListKindS = map[ast.NestedListKind]*sxpf.Symbol{
		ast.NestedListOrdered:   t.zetSyms.SymListOrdered,
		ast.NestedListUnordered: t.zetSyms.SymListUnordered,
		ast.NestedListQuote:     t.zetSyms.SymListQuote,
	}
	t.alignmentSymbolS = map[ast.Alignment]*sxpf.Symbol{
		ast.AlignDefault: t.zetSyms.SymCell,
		ast.AlignLeft:    t.zetSyms.SymCellLeft,
		ast.AlignCenter:  t.zetSyms.SymCellCenter,
		ast.AlignRight:   t.zetSyms.SymCellRight,
	}
	t.mapRefStateLink = map[ast.RefState]*sxpf.Symbol{
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
	t.mapFormatKindS = map[ast.FormatKind]*sxpf.Symbol{
		ast.FormatEmph:   t.zetSyms.SymFormatEmph,
		ast.FormatStrong: t.zetSyms.SymFormatStrong,
		ast.FormatDelete: t.zetSyms.SymFormatDelete,
		ast.FormatInsert: t.zetSyms.SymFormatInsert,
		ast.FormatSuper:  t.zetSyms.SymFormatSuper,
		ast.FormatSub:    t.zetSyms.SymFormatSub,
		ast.FormatQuote:  t.zetSyms.SymFormatQuote,
		ast.FormatSpan:   t.zetSyms.SymFormatSpan,
	}
	t.mapLiteralKindS = map[ast.LiteralKind]*sxpf.Symbol{
		ast.LiteralZettel:  t.zetSyms.SymLiteralZettel,
		ast.LiteralProg:    t.zetSyms.SymLiteralProg,
		ast.LiteralInput:   t.zetSyms.SymLiteralInput,
		ast.LiteralOutput:  t.zetSyms.SymLiteralOutput,
		ast.LiteralComment: t.zetSyms.SymLiteralComment,
		ast.LiteralHTML:    t.zetSyms.SymLiteralHTML,
		ast.LiteralMath:    t.zetSyms.SymLiteralMath,
	}
	t.mapRefStateS = map[ast.RefState]*sxpf.Symbol{
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
	t.mapMetaTypeS = map[*meta.DescriptionType]*sxpf.Symbol{
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
	sf                 sxpf.SymbolFactory
	zetSyms            sexpr.ZettelSymbols
	mapVerbatimKindS   map[ast.VerbatimKind]*sxpf.Symbol
	mapRegionKindS     map[ast.RegionKind]*sxpf.Symbol
	mapNestedListKindS map[ast.NestedListKind]*sxpf.Symbol
	alignmentSymbolS   map[ast.Alignment]*sxpf.Symbol
	mapRefStateLink    map[ast.RefState]*sxpf.Symbol
	mapFormatKindS     map[ast.FormatKind]*sxpf.Symbol
	mapLiteralKindS    map[ast.LiteralKind]*sxpf.Symbol
	mapRefStateS       map[ast.RefState]*sxpf.Symbol
	mapMetaTypeS       map[*meta.DescriptionType]*sxpf.Symbol
	inVerse            bool
}

func (t *Transformer) GetSexpr(node ast.Node) *sxpf.List {
	switch n := node.(type) {
	case *ast.BlockSlice:
		return t.getBlockSlice(n)
	case *ast.InlineSlice:
		return t.getInlineSlice(*n)
	case *ast.ParaNode:
		return t.getInlineSlice(n.Inlines).Tail().Cons(t.zetSyms.SymPara)
	case *ast.VerbatimNode:
		return sxpf.MakeList(
			mapGetS(t, t.mapVerbatimKindS, n.Kind),
			t.getAttributes(n.Attrs),
			sxpf.MakeString(string(n.Content)),
		)
	case *ast.RegionNode:
		return t.getRegion(n)
	case *ast.HeadingNode:
		return t.getInlineSlice(n.Inlines).Tail().
			Cons(sxpf.MakeString(n.Fragment)).
			Cons(sxpf.MakeString(n.Slug)).
			Cons(t.getAttributes(n.Attrs)).
			Cons(sxpf.MakeInteger64(int64(n.Level))).
			Cons(t.zetSyms.SymHeading)
	case *ast.HRuleNode:
		return sxpf.MakeList(t.zetSyms.SymThematic, t.getAttributes(n.Attrs))
	case *ast.NestedListNode:
		return t.getNestedList(n)
	case *ast.DescriptionListNode:
		return t.getDescriptionList(n)
	case *ast.TableNode:
		return t.getTable(n)
	case *ast.TranscludeNode:
		return sxpf.MakeList(t.zetSyms.SymTransclude, t.getAttributes(n.Attrs), t.getReference(n.Ref))
	case *ast.BLOBNode:
		return t.getBLOB(n)
	case *ast.TextNode:
		return sxpf.MakeList(t.zetSyms.SymText, sxpf.MakeString(n.Text))
	case *ast.SpaceNode:
		if t.inVerse {
			return sxpf.MakeList(t.zetSyms.SymSpace, sxpf.MakeString(n.Lexeme))
		}
		return sxpf.MakeList(t.zetSyms.SymSpace)
	case *ast.BreakNode:
		if n.Hard {
			return sxpf.MakeList(t.zetSyms.SymHard)
		}
		return sxpf.MakeList(t.zetSyms.SymSoft)
	case *ast.LinkNode:
		return t.getLink(n)
	case *ast.EmbedRefNode:
		return t.getInlineSlice(n.Inlines).Tail().
			Cons(sxpf.MakeString(n.Syntax)).
			Cons(t.getReference(n.Ref)).
			Cons(t.getAttributes(n.Attrs)).
			Cons(t.zetSyms.SymEmbed)
	case *ast.EmbedBLOBNode:
		return t.getEmbedBLOB(n)
	case *ast.CiteNode:
		return t.getInlineSlice(n.Inlines).Tail().
			Cons(sxpf.MakeString(n.Key)).
			Cons(t.getAttributes(n.Attrs)).
			Cons(t.zetSyms.SymCite)
	case *ast.FootnoteNode:
		return t.getInlineSlice(n.Inlines).Tail().Cons(t.getAttributes(n.Attrs)).Cons(t.zetSyms.SymEndnote)
	case *ast.MarkNode:
		return t.getInlineSlice(n.Inlines).Tail().
			Cons(sxpf.MakeString(n.Fragment)).
			Cons(sxpf.MakeString(n.Slug)).
			Cons(sxpf.MakeString(n.Mark)).
			Cons(t.zetSyms.SymMark)
	case *ast.FormatNode:
		return t.getInlineSlice(n.Inlines).Tail().
			Cons(t.getAttributes(n.Attrs)).
			Cons(mapGetS(t, t.mapFormatKindS, n.Kind))
	case *ast.LiteralNode:
		return sxpf.MakeList(
			mapGetS(t, t.mapLiteralKindS, n.Kind),
			t.getAttributes(n.Attrs),
			sxpf.MakeString(string(n.Content)),
		)
	}
	log.Printf("SEXPR %T %v\n", node, node)
	return sxpf.MakeList(t.zetSyms.SymUnknown, sxpf.MakeString(fmt.Sprintf("%T %v", node, node)))
}

func (t *Transformer) getRegion(rn *ast.RegionNode) *sxpf.List {
	saveInVerse := t.inVerse
	if rn.Kind == ast.RegionVerse {
		t.inVerse = true
	}
	symBlocks := t.GetSexpr(&rn.Blocks)
	t.inVerse = saveInVerse
	return sxpf.MakeList(
		mapGetS(t, t.mapRegionKindS, rn.Kind),
		t.getAttributes(rn.Attrs),
		symBlocks,
		t.GetSexpr(&rn.Inlines),
	)
}

func (t *Transformer) getNestedList(ln *ast.NestedListNode) *sxpf.List {
	nlistObjs := make([]sxpf.Object, len(ln.Items)+1)
	nlistObjs[0] = mapGetS(t, t.mapNestedListKindS, ln.Kind)
	isCompact := isCompactList(ln.Items)
	for i, item := range ln.Items {
		if isCompact && len(item) > 0 {
			paragraph := t.GetSexpr(item[0])
			nlistObjs[i+1] = paragraph.Tail().Cons(t.zetSyms.SymInline)
			continue
		}
		itemObjs := make([]sxpf.Object, len(item))
		for j, in := range item {
			itemObjs[j] = t.GetSexpr(in)
		}
		if isCompact {
			nlistObjs[i+1] = sxpf.MakeList(itemObjs...).Cons(t.zetSyms.SymInline)
		} else {
			nlistObjs[i+1] = sxpf.MakeList(itemObjs...).Cons(t.zetSyms.SymBlock)
		}
	}
	return sxpf.MakeList(nlistObjs...)
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

func (t *Transformer) getDescriptionList(dn *ast.DescriptionListNode) *sxpf.List {
	dlObjs := make([]sxpf.Object, 2*len(dn.Descriptions)+1)
	dlObjs[0] = t.zetSyms.SymDescription
	for i, def := range dn.Descriptions {
		dlObjs[2*i+1] = t.getInlineSlice(def.Term)
		descObjs := make([]sxpf.Object, len(def.Descriptions))
		for j, b := range def.Descriptions {
			dVal := make([]sxpf.Object, len(b))
			for k, dn := range b {
				dVal[k] = t.GetSexpr(dn)
			}
			descObjs[j] = sxpf.MakeList(dVal...).Cons(t.zetSyms.SymBlock)
		}
		dlObjs[2*i+2] = sxpf.MakeList(descObjs...).Cons(t.zetSyms.SymBlock)
	}
	return sxpf.MakeList(dlObjs...)
}

func (t *Transformer) getTable(tn *ast.TableNode) *sxpf.List {
	tObjs := make([]sxpf.Object, len(tn.Rows)+2)
	tObjs[0] = t.zetSyms.SymTable
	tObjs[1] = t.getHeader(tn.Header)
	for i, row := range tn.Rows {
		tObjs[i+2] = t.getRow(row)
	}
	return sxpf.MakeList(tObjs...)
}
func (t *Transformer) getHeader(header ast.TableRow) *sxpf.List {
	if len(header) == 0 {
		return sxpf.Nil()
	}
	return t.getRow(header)
}
func (t *Transformer) getRow(row ast.TableRow) *sxpf.List {
	rObjs := make([]sxpf.Object, len(row))
	for i, cell := range row {
		rObjs[i] = t.getCell(cell)
	}
	return sxpf.MakeList(rObjs...).Cons(t.zetSyms.SymRow)
}

func (t *Transformer) getCell(cell *ast.TableCell) *sxpf.List {
	return t.getInlineSlice(cell.Inlines).Tail().Cons(mapGetS(t, t.alignmentSymbolS, cell.Align))
}

func (t *Transformer) getBLOB(bn *ast.BLOBNode) *sxpf.List {
	var lastObj sxpf.Object
	if bn.Syntax == meta.SyntaxSVG {
		lastObj = sxpf.MakeString(string(bn.Blob))
	} else {
		lastObj = getBase64String(bn.Blob)
	}
	return sxpf.MakeList(
		t.zetSyms.SymBLOB,
		t.getInlineSlice(bn.Description),
		sxpf.MakeString(bn.Syntax),
		lastObj,
	)
}

func (t *Transformer) getLink(ln *ast.LinkNode) *sxpf.List {
	return t.getInlineSlice(ln.Inlines).Tail().
		Cons(sxpf.MakeString(ln.Ref.Value)).
		Cons(t.getAttributes(ln.Attrs)).
		Cons(mapGetS(t, t.mapRefStateLink, ln.Ref.State))
}

func (t *Transformer) getEmbedBLOB(en *ast.EmbedBLOBNode) *sxpf.List {
	tail := t.getInlineSlice(en.Inlines).Tail()
	if en.Syntax == meta.SyntaxSVG {
		tail = tail.Cons(sxpf.MakeString(string(en.Blob)))
	} else {
		tail = tail.Cons(getBase64String(en.Blob))
	}
	return tail.Cons(sxpf.MakeString(en.Syntax)).Cons(t.getAttributes(en.Attrs)).Cons(t.zetSyms.SymEmbedBLOB)
}

func (t *Transformer) getBlockSlice(bs *ast.BlockSlice) *sxpf.List {
	objs := make([]sxpf.Object, len(*bs))
	for i, n := range *bs {
		objs[i] = t.GetSexpr(n)
	}
	return sxpf.MakeList(objs...).Cons(t.zetSyms.SymBlock)
}
func (t *Transformer) getInlineSlice(is ast.InlineSlice) *sxpf.List {
	objs := make([]sxpf.Object, len(is))
	for i, n := range is {
		objs[i] = t.GetSexpr(n)
	}
	return sxpf.MakeList(objs...).Cons(t.zetSyms.SymInline)
}

func (t *Transformer) getAttributes(a attrs.Attributes) sxpf.Object {
	if a.IsEmpty() {
		return sxpf.Nil()
	}
	keys := a.Keys()
	objs := make([]sxpf.Object, 0, len(keys))
	for _, k := range keys {
		objs = append(objs, sxpf.Cons(sxpf.MakeString(k), sxpf.MakeString(a[k])))
	}
	return sxpf.MakeList(objs...).Cons(t.zetSyms.SymQuote)
}

func (t *Transformer) getReference(ref *ast.Reference) *sxpf.List {
	return sxpf.Nil().Cons(sxpf.MakeString(ref.Value)).Cons(mapGetS(t, t.mapRefStateS, ref.State))
}

func (t *Transformer) GetMeta(m *meta.Meta, evalMeta encoder.EvalMetaFunc) *sxpf.List {
	pairs := m.ComputedPairs()
	objs := make([]sxpf.Object, 0, len(pairs))
	for _, p := range pairs {
		key := p.Key
		ty := m.Type(key)
		symType := mapGetS(t, t.mapMetaTypeS, ty)
		strKey := sxpf.MakeString(key)
		var obj sxpf.Object
		if ty.IsSet {
			setList := meta.ListFromValue(p.Value)
			setObjs := make([]sxpf.Object, len(setList))
			for i, val := range setList {
				setObjs[i] = sxpf.MakeString(val)
			}
			obj = sxpf.MakeList(setObjs...)
		} else if ty == meta.TypeZettelmarkup {
			is := evalMeta(p.Value)
			obj = t.GetSexpr(&is)
		} else {
			obj = sxpf.MakeString(p.Value)
		}
		objs = append(objs, sxpf.Nil().Cons(obj).Cons(strKey).Cons(symType))
	}
	return sxpf.MakeList(objs...).Cons(t.zetSyms.SymMeta)
}

func mapGetS[T comparable](t *Transformer, m map[T]*sxpf.Symbol, k T) *sxpf.Symbol {
	if result, found := m[k]; found {
		return result
	}
	log.Println("MISS", k, m)
	return t.sf.Make(fmt.Sprintf("**%v:NOT-FOUND**", k))
}

func getBase64String(data []byte) sxpf.String {
	var sb strings.Builder
	encoder := base64.NewEncoder(base64.StdEncoding, &sb)
	_, err := encoder.Write(data)
	if err == nil {
		err = encoder.Close()
	}
	if err == nil {
		return sxpf.MakeString(sb.String())
	}
	return sxpf.MakeString("")
}
