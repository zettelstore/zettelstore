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
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"strconv"

	"github.com/t73fde/sxpf"
	"zettelstore.de/c/api"
	"zettelstore.de/c/attrs"
	"zettelstore.de/c/sexpr"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
)

// GetSexpr returns the given node as a s-expression.
func GetSexpr(node ast.Node) *sxpf.List {
	t := transformer{}
	return t.getSexpr(node)
}

type transformer struct {
	inVerse bool
}

func (t *transformer) getSexpr(node ast.Node) *sxpf.List {
	switch n := node.(type) {
	case *ast.BlockSlice:
		return t.getBlockSlice(n)
	case *ast.InlineSlice:
		return t.getInlineSlice(*n)
	case *ast.ParaNode:
		result := sxpf.NewList(sexpr.SymPara)
		result.Extend(t.getInlineSlice(n.Inlines))
		return result
	case *ast.VerbatimNode:
		return sxpf.NewList(
			mapGetS(mapVerbatimKindS, n.Kind),
			getAttributes(n.Attrs),
			sxpf.NewString(string(n.Content)),
		)
	case *ast.RegionNode:
		return t.getRegion(n)
	case *ast.HeadingNode:
		result := sxpf.NewList(
			sexpr.SymHeading,
			sxpf.NewSymbol(strconv.Itoa(n.Level)),
			getAttributes(n.Attrs),
			sxpf.NewString(n.Slug),
			sxpf.NewString(n.Fragment),
		)
		result.Extend(t.getInlineSlice(n.Inlines))
		return result
	case *ast.HRuleNode:
		return sxpf.NewList(sexpr.SymThematic, getAttributes(n.Attrs))
	case *ast.NestedListNode:
		return t.getNestedList(n)
	case *ast.DescriptionListNode:
		return t.getDescriptionList(n)
	case *ast.TableNode:
		return t.getTable(n)
	case *ast.TranscludeNode:
		return sxpf.NewList(sexpr.SymTransclude, getReference(n.Ref))
	case *ast.BLOBNode:
		return getBLOB(n)
	case *ast.TextNode:
		return sxpf.NewList(sexpr.SymText, sxpf.NewString(n.Text))
	case *ast.TagNode:
		return sxpf.NewList(sexpr.SymTag, sxpf.NewString(n.Tag))
	case *ast.SpaceNode:
		result := sxpf.NewList(sexpr.SymSpace)
		if t.inVerse {
			result.Append(sxpf.NewString(n.Lexeme))
		}
		return result
	case *ast.BreakNode:
		if n.Hard {
			return sxpf.NewList(sexpr.SymHard)
		} else {
			return sxpf.NewList(sexpr.SymSoft)
		}
	case *ast.LinkNode:
		result := sxpf.NewList(
			sexpr.SymLink,
			getAttributes(n.Attrs),
			getReference(n.Ref),
		)
		result.Extend(t.getInlineSlice(n.Inlines))
		return result
	case *ast.EmbedRefNode:
		result := sxpf.NewList(
			sexpr.SymEmbed,
			getAttributes(n.Attrs),
			getReference(n.Ref),
			sxpf.NewString(n.Syntax),
		)
		result.Extend(t.getInlineSlice(n.Inlines))
		return result
	case *ast.EmbedBLOBNode:
		return t.getEmbedBLOB(n)
	case *ast.CiteNode:
		result := sxpf.NewList(
			sexpr.SymCite,
			getAttributes(n.Attrs),
			sxpf.NewString(n.Key),
		)
		result.Extend(t.getInlineSlice(n.Inlines))
		return result
	case *ast.FootnoteNode:
		result := sxpf.NewList(sexpr.SymFootnote, getAttributes(n.Attrs))
		result.Extend(t.getInlineSlice(n.Inlines))
		return result
	case *ast.MarkNode:
		result := sxpf.NewList(
			sexpr.SymMark,
			sxpf.NewString(n.Mark),
			sxpf.NewString(n.Slug),
			sxpf.NewString(n.Fragment),
		)
		result.Extend(t.getInlineSlice(n.Inlines))
		return result
	case *ast.FormatNode:
		result := sxpf.NewList(mapGetS(mapFormatKindS, n.Kind), getAttributes(n.Attrs))
		result.Extend(t.getInlineSlice(n.Inlines))
		return result
	case *ast.LiteralNode:
		return sxpf.NewList(
			mapGetS(mapLiteralKindS, n.Kind),
			getAttributes(n.Attrs),
			sxpf.NewString(string(n.Content)),
		)
	}
	log.Printf("SEXPR %T %v\n", node, node)
	return sxpf.NewList(sexpr.SymUnknown, sxpf.NewString(fmt.Sprintf("%T %v", node, node)))
}

var mapVerbatimKindS = map[ast.VerbatimKind]*sxpf.Symbol{
	ast.VerbatimZettel:  sexpr.SymVerbatimZettel,
	ast.VerbatimProg:    sexpr.SymVerbatimProg,
	ast.VerbatimEval:    sexpr.SymVerbatimEval,
	ast.VerbatimMath:    sexpr.SymVerbatimMath,
	ast.VerbatimComment: sexpr.SymVerbatimComment,
	ast.VerbatimHTML:    sexpr.SymVerbatimHTML,
}

var mapRegionKindS = map[ast.RegionKind]*sxpf.Symbol{
	ast.RegionSpan:  sexpr.SymRegionBlock,
	ast.RegionQuote: sexpr.SymRegionQuote,
	ast.RegionVerse: sexpr.SymRegionVerse,
}

func (t *transformer) getRegion(rn *ast.RegionNode) *sxpf.List {
	saveInVerse := t.inVerse
	if rn.Kind == ast.RegionVerse {
		t.inVerse = true
	}
	symBlocks := t.getSexpr(&rn.Blocks)
	t.inVerse = saveInVerse
	return sxpf.NewList(
		mapGetS(mapRegionKindS, rn.Kind),
		getAttributes(rn.Attrs),
		symBlocks,
		t.getSexpr(&rn.Inlines),
	)
}

var mapNestedListKindS = map[ast.NestedListKind]*sxpf.Symbol{
	ast.NestedListOrdered:   sexpr.SymListOrdered,
	ast.NestedListUnordered: sexpr.SymListUnordered,
	ast.NestedListQuote:     sexpr.SymListQuote,
}

func (t *transformer) getNestedList(ln *ast.NestedListNode) *sxpf.List {
	nlistVals := make([]sxpf.Value, len(ln.Items)+1)
	nlistVals[0] = mapGetS(mapNestedListKindS, ln.Kind)
	for i, item := range ln.Items {
		itemVals := make([]sxpf.Value, len(item))
		for j, in := range item {
			itemVals[j] = t.getSexpr(in)
		}
		nlistVals[i+1] = sxpf.NewList(itemVals...)
	}
	return sxpf.NewList(nlistVals...)
}

func (t *transformer) getDescriptionList(dn *ast.DescriptionListNode) *sxpf.List {
	dlVals := make([]sxpf.Value, 2*len(dn.Descriptions)+1)
	dlVals[0] = sexpr.SymDescription
	for i, def := range dn.Descriptions {
		dlVals[2*i+1] = t.getInlineSlice(def.Term)
		descVals := make([]sxpf.Value, len(def.Descriptions))
		for j, b := range def.Descriptions {
			dVal := make([]sxpf.Value, len(b))
			for k, dn := range b {
				dVal[k] = t.getSexpr(dn)
			}
			descVals[j] = sxpf.NewList(dVal...)
		}
		dlVals[2*i+2] = sxpf.NewList(descVals...)
	}
	return sxpf.NewList(dlVals...)
}

func (t *transformer) getTable(tn *ast.TableNode) *sxpf.List {
	tVals := make([]sxpf.Value, len(tn.Rows)+2)
	tVals[0] = sexpr.SymTable
	tVals[1] = t.getRow(tn.Header)
	for i, row := range tn.Rows {
		tVals[i+2] = t.getRow(row)
	}
	return sxpf.NewList(tVals...)
}
func (t *transformer) getRow(row ast.TableRow) *sxpf.List {
	rVals := make([]sxpf.Value, len(row))
	for i, cell := range row {
		rVals[i] = t.getCell(cell)
	}
	return sxpf.NewList(rVals...)
}

var alignmentSymbolS = map[ast.Alignment]*sxpf.Symbol{
	ast.AlignDefault: sexpr.SymCell,
	ast.AlignLeft:    sexpr.SymCellLeft,
	ast.AlignCenter:  sexpr.SymCellCenter,
	ast.AlignRight:   sexpr.SymCellRight,
}

func (t *transformer) getCell(cell *ast.TableCell) *sxpf.List {
	result := sxpf.NewList(mapGetS(alignmentSymbolS, cell.Align))
	result.Extend(t.getInlineSlice(cell.Inlines))
	return result
}

func getBLOB(bn *ast.BLOBNode) *sxpf.List {
	result := sxpf.NewList(
		sexpr.SymBLOB,
		sxpf.NewString(bn.Title),
		sxpf.NewString(bn.Syntax),
	)
	if bn.Syntax == api.ValueSyntaxSVG {
		result.Append(sxpf.NewString(string(bn.Blob)))
	} else {
		result.Append(getBase64String(bn.Blob))
	}
	return result
}

func (t *transformer) getEmbedBLOB(en *ast.EmbedBLOBNode) *sxpf.List {
	result := sxpf.NewList(
		sexpr.SymEmbedBLOB,
		getAttributes(en.Attrs),
		sxpf.NewString(en.Syntax),
	)
	if en.Syntax == api.ValueSyntaxSVG {
		result.Append(sxpf.NewString(string(en.Blob)))
	} else {
		result.Append(getBase64String(en.Blob))
	}
	result.Extend(t.getInlineSlice(en.Inlines))
	return result
}

var mapFormatKindS = map[ast.FormatKind]*sxpf.Symbol{
	ast.FormatEmph:   sexpr.SymFormatEmph,
	ast.FormatStrong: sexpr.SymFormatStrong,
	ast.FormatDelete: sexpr.SymFormatDelete,
	ast.FormatInsert: sexpr.SymFormatInsert,
	ast.FormatSuper:  sexpr.SymFormatSuper,
	ast.FormatSub:    sexpr.SymFormatSub,
	ast.FormatQuote:  sexpr.SymFormatQuote,
	ast.FormatSpan:   sexpr.SymFormatSpan,
}

var mapLiteralKindS = map[ast.LiteralKind]*sxpf.Symbol{
	ast.LiteralZettel:  sexpr.SymLiteralZettel,
	ast.LiteralProg:    sexpr.SymLiteralProg,
	ast.LiteralInput:   sexpr.SymLiteralInput,
	ast.LiteralOutput:  sexpr.SymLiteralOutput,
	ast.LiteralComment: sexpr.SymLiteralComment,
	ast.LiteralHTML:    sexpr.SymLiteralHTML,
	ast.LiteralMath:    sexpr.SymLiteralMath,
}

func (t *transformer) getBlockSlice(bs *ast.BlockSlice) *sxpf.List {
	lstVals := make([]sxpf.Value, len(*bs))
	for i, n := range *bs {
		lstVals[i] = t.getSexpr(n)
	}
	return sxpf.NewList(lstVals...)
}
func (t *transformer) getInlineSlice(is ast.InlineSlice) *sxpf.List {
	lstVals := make([]sxpf.Value, len(is))
	for i, n := range is {
		lstVals[i] = t.getSexpr(n)
	}
	return sxpf.NewList(lstVals...)
}

func getAttributes(a attrs.Attributes) *sxpf.List {
	if a.IsEmpty() {
		return sxpf.NewList()
	}
	keys := a.Keys()
	lstVals := make([]sxpf.Value, 0, len(keys))
	for _, k := range keys {
		lstVals = append(lstVals, sxpf.NewList(sxpf.NewString(k), sxpf.NewString(a[k])))
	}
	return sxpf.NewList(lstVals...)
}

var mapRefStateS = map[ast.RefState]*sxpf.Symbol{
	ast.RefStateInvalid:  sexpr.SymRefStateInvalid,
	ast.RefStateZettel:   sexpr.SymRefStateZettel,
	ast.RefStateSelf:     sexpr.SymRefStateSelf,
	ast.RefStateFound:    sexpr.SymRefStateFound,
	ast.RefStateBroken:   sexpr.SymRefStateBroken,
	ast.RefStateHosted:   sexpr.SymRefStateHosted,
	ast.RefStateBased:    sexpr.SymRefStateBased,
	ast.RefStateExternal: sexpr.SymRefStateExternal,
}

func getReference(ref *ast.Reference) *sxpf.List {
	return sxpf.NewList(mapGetS(mapRefStateS, ref.State), sxpf.NewString(ref.Value))
}

var mapMetaTypeS = map[*meta.DescriptionType]*sxpf.Symbol{
	meta.TypeCredential:   sexpr.SymTypeCredential,
	meta.TypeEmpty:        sexpr.SymTypeEmpty,
	meta.TypeID:           sexpr.SymTypeID,
	meta.TypeIDSet:        sexpr.SymTypeIDSet,
	meta.TypeNumber:       sexpr.SymTypeNumber,
	meta.TypeString:       sexpr.SymTypeString,
	meta.TypeTagSet:       sexpr.SymTypeTagSet,
	meta.TypeTimestamp:    sexpr.SymTypeTimestamp,
	meta.TypeURL:          sexpr.SymTypeURL,
	meta.TypeWord:         sexpr.SymTypeWord,
	meta.TypeWordSet:      sexpr.SymTypeWordSet,
	meta.TypeZettelmarkup: sexpr.SymTypeZettelmarkup,
}

func GetMeta(m *meta.Meta, evalMeta encoder.EvalMetaFunc) *sxpf.List {
	pairs := m.ComputedPairs()
	lstVals := make([]sxpf.Value, 0, len(pairs))
	for _, p := range pairs {
		key := p.Key
		ty := m.Type(key)
		symType := mapGetS(mapMetaTypeS, ty)
		symKey := sxpf.NewSymbol(key)
		var val sxpf.Value
		if ty.IsSet {
			setList := meta.ListFromValue(p.Value)
			setVals := make([]sxpf.Value, len(setList))
			for i, val := range setList {
				setVals[i] = sxpf.NewString(val)
			}
			val = sxpf.NewList(setVals...)
		} else if ty == meta.TypeZettelmarkup {
			is := evalMeta(p.Value)
			t := transformer{}
			val = t.getSexpr(&is)
		} else {
			val = sxpf.NewString(p.Value)
		}
		lstVals = append(lstVals, sxpf.NewList(symType, symKey, val))
	}
	return sxpf.NewList(lstVals...)
}

func mapGetS[T comparable](m map[T]*sxpf.Symbol, k T) *sxpf.Symbol {
	if result, found := m[k]; found {
		return result
	}
	log.Println("MISS", k, m)
	return sxpf.NewSymbol(fmt.Sprintf("**%v:not-found**", k))
}

func getBase64String(data []byte) *sxpf.String {
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	_, err := encoder.Write(data)
	if err == nil {
		err = encoder.Close()
	}
	if err == nil {
		return sxpf.NewString(buf.String())
	}
	return nil
}
