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

	"zettelstore.de/c/api"
	"zettelstore.de/c/sexpr"
	"zettelstore.de/c/zjson"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
)

type transformer struct {
	inVerse bool
}

func (t *transformer) getSexpr(node ast.Node) *sexpr.List {
	switch n := node.(type) {
	case *ast.BlockSlice:
		return t.getBlockSlice(n)
	case *ast.InlineSlice:
		return t.getInlineSlice(*n)
	case *ast.ParaNode:
		result := sexpr.NewList(sexpr.SymPara)
		result.Extend(t.getInlineSlice(n.Inlines))
		return result
	case *ast.VerbatimNode:
		return sexpr.NewList(
			mapGetS(mapVerbatimKindS, n.Kind),
			getAttributes(n.Attrs),
			sexpr.NewString(string(n.Content)),
		)
	case *ast.RegionNode:
		return t.getRegion(n)
	case *ast.HeadingNode:
		result := sexpr.NewList(
			sexpr.SymHeading,
			sexpr.GetSymbol(strconv.Itoa(n.Level)),
			getAttributes(n.Attrs),
			sexpr.NewString(n.Slug),
			sexpr.NewString(n.Fragment),
		)
		result.Extend(t.getInlineSlice(n.Inlines))
		return result
	case *ast.HRuleNode:
		return sexpr.NewList(sexpr.SymThematic, getAttributes(n.Attrs))
	case *ast.NestedListNode:
		return t.getNestedList(n)
	case *ast.DescriptionListNode:
		return t.getDescriptionList(n)
	case *ast.TableNode:
		return t.getTable(n)
	case *ast.TranscludeNode:
		return sexpr.NewList(sexpr.SymTransclude, getReference(n.Ref))
	case *ast.BLOBNode:
		return getBLOB(n)
	case *ast.TextNode:
		return sexpr.NewList(sexpr.SymText, sexpr.NewString(n.Text))
	case *ast.TagNode:
		return sexpr.NewList(sexpr.SymTag, sexpr.NewString(n.Tag))
	case *ast.SpaceNode:
		result := sexpr.NewList(sexpr.SymSpace)
		if t.inVerse {
			result.Append(sexpr.NewString(n.Lexeme))
		}
		return result
	case *ast.BreakNode:
		if n.Hard {
			return sexpr.NewList(sexpr.SymHard)
		} else {
			return sexpr.NewList(sexpr.SymSoft)
		}
	case *ast.LinkNode:
		result := sexpr.NewList(
			sexpr.SymLink,
			getAttributes(n.Attrs),
			getReference(n.Ref),
		)
		result.Extend(t.getInlineSlice(n.Inlines))
		return result
	case *ast.EmbedRefNode:
		result := sexpr.NewList(
			sexpr.SymEmbed,
			getAttributes(n.Attrs),
			getReference(n.Ref),
			sexpr.NewString(n.Syntax),
		)
		result.Extend(t.getInlineSlice(n.Inlines))
		return result
	case *ast.EmbedBLOBNode:
		return t.getEmbedBLOB(n)
	case *ast.CiteNode:
		result := sexpr.NewList(
			sexpr.SymCite,
			getAttributes(n.Attrs),
			sexpr.NewString(n.Key),
		)
		result.Extend(t.getInlineSlice(n.Inlines))
		return result
	case *ast.FootnoteNode:
		result := sexpr.NewList(sexpr.SymFootnote, getAttributes(n.Attrs))
		result.Extend(t.getInlineSlice(n.Inlines))
		return result
	case *ast.MarkNode:
		result := sexpr.NewList(
			sexpr.SymMark,
			sexpr.NewString(n.Mark),
			sexpr.NewString(n.Slug),
			sexpr.NewString(n.Fragment),
		)
		result.Extend(t.getInlineSlice(n.Inlines))
		return result
	case *ast.FormatNode:
		result := sexpr.NewList(mapGetS(mapFormatKindS, n.Kind), getAttributes(n.Attrs))
		result.Extend(t.getInlineSlice(n.Inlines))
		return result
	case *ast.LiteralNode:
		return sexpr.NewList(
			mapGetS(mapLiteralKindS, n.Kind),
			getAttributes(n.Attrs),
			sexpr.NewString(string(n.Content)),
		)
	}
	log.Printf("SEXPR %T %v\n", node, node)
	return sexpr.NewList(sexpr.SymUnknown, sexpr.NewString(fmt.Sprintf("%T %v", node, node)))
}

var mapVerbatimKindS = map[ast.VerbatimKind]*sexpr.Symbol{
	ast.VerbatimZettel:  sexpr.SymVerbatimZettel,
	ast.VerbatimProg:    sexpr.SymVerbatimProg,
	ast.VerbatimEval:    sexpr.SymVerbatimEval,
	ast.VerbatimMath:    sexpr.SymVerbatimMath,
	ast.VerbatimComment: sexpr.SymVerbatimComment,
	ast.VerbatimHTML:    sexpr.SymVerbatimHTML,
}

var mapRegionKindS = map[ast.RegionKind]*sexpr.Symbol{
	ast.RegionSpan:  sexpr.SymRegionSpan,
	ast.RegionQuote: sexpr.SymRegionQuote,
	ast.RegionVerse: sexpr.SymRegionVerse,
}

func (t *transformer) getRegion(rn *ast.RegionNode) *sexpr.List {
	saveInVerse := t.inVerse
	if rn.Kind == ast.RegionVerse {
		t.inVerse = true
	}
	symBlocks := t.getSexpr(&rn.Blocks)
	t.inVerse = saveInVerse
	return sexpr.NewList(
		mapGetS(mapRegionKindS, rn.Kind),
		getAttributes(rn.Attrs),
		symBlocks,
		t.getSexpr(&rn.Inlines),
	)
}

var mapNestedListKindS = map[ast.NestedListKind]*sexpr.Symbol{
	ast.NestedListOrdered:   sexpr.SymListOrdered,
	ast.NestedListUnordered: sexpr.SymListUnordered,
	ast.NestedListQuote:     sexpr.SymListQuote,
}

func (t *transformer) getNestedList(ln *ast.NestedListNode) *sexpr.List {
	nlistVals := make([]sexpr.Value, len(ln.Items)+1)
	nlistVals[0] = mapGetS(mapNestedListKindS, ln.Kind)
	for i, item := range ln.Items {
		itemVals := make([]sexpr.Value, len(item))
		for j, in := range item {
			itemVals[j] = t.getSexpr(in)
		}
		nlistVals[i+1] = sexpr.NewList(itemVals...)
	}
	return sexpr.NewList(nlistVals...)
}

func (t *transformer) getDescriptionList(dn *ast.DescriptionListNode) *sexpr.List {
	dlVals := make([]sexpr.Value, 2*len(dn.Descriptions)+1)
	dlVals[0] = sexpr.SymDescription
	for i, def := range dn.Descriptions {
		dlVals[2*i+1] = t.getInlineSlice(def.Term)
		descVals := make([]sexpr.Value, len(def.Descriptions))
		for j, b := range def.Descriptions {
			dVal := make([]sexpr.Value, len(b))
			for k, dn := range b {
				dVal[k] = t.getSexpr(dn)
			}
			descVals[j] = sexpr.NewList(dVal...)
		}
		dlVals[2*i+2] = sexpr.NewList(descVals...)
	}
	return sexpr.NewList(dlVals...)
}

func (t *transformer) getTable(tn *ast.TableNode) *sexpr.List {
	tVals := make([]sexpr.Value, len(tn.Rows)+2)
	tVals[0] = sexpr.SymTable
	tVals[1] = t.getRow(tn.Header)
	for i, row := range tn.Rows {
		tVals[i+2] = t.getRow(row)
	}
	return sexpr.NewList(tVals...)
}
func (t *transformer) getRow(row ast.TableRow) *sexpr.List {
	rVals := make([]sexpr.Value, len(row))
	for i, cell := range row {
		rVals[i] = t.getCell(cell)
	}
	return sexpr.NewList(rVals...)
}

var alignmentSymbolS = map[ast.Alignment]*sexpr.Symbol{
	ast.AlignDefault: sexpr.SymCell,
	ast.AlignLeft:    sexpr.SymCellLeft,
	ast.AlignCenter:  sexpr.SymCellCenter,
	ast.AlignRight:   sexpr.SymCellRight,
}

func (t *transformer) getCell(cell *ast.TableCell) *sexpr.List {
	result := sexpr.NewList(mapGetS(alignmentSymbolS, cell.Align))
	result.Extend(t.getInlineSlice(cell.Inlines))
	return result
}

func getBLOB(bn *ast.BLOBNode) *sexpr.List {
	result := sexpr.NewList(
		sexpr.SymBLOB,
		sexpr.NewString(bn.Title),
		sexpr.NewString(bn.Syntax),
	)
	if bn.Syntax == api.ValueSyntaxSVG {
		result.Append(sexpr.NewString(string(bn.Blob)))
	} else {
		result.Append(getBase64String(bn.Blob))
	}
	return result
}

func (t *transformer) getEmbedBLOB(en *ast.EmbedBLOBNode) *sexpr.List {
	result := sexpr.NewList(
		sexpr.SymEmbedBLOB,
		getAttributes(en.Attrs),
		sexpr.NewString(en.Syntax),
	)
	if en.Syntax == api.ValueSyntaxSVG {
		result.Append(sexpr.NewString(string(en.Blob)))
	} else {
		result.Append(getBase64String(en.Blob))
	}
	result.Extend(t.getInlineSlice(en.Inlines))
	return result
}

var mapFormatKindS = map[ast.FormatKind]*sexpr.Symbol{
	ast.FormatEmph:   sexpr.SymFormatEmph,
	ast.FormatStrong: sexpr.SymFormatStrong,
	ast.FormatDelete: sexpr.SymFormatDelete,
	ast.FormatInsert: sexpr.SymFormatInsert,
	ast.FormatSuper:  sexpr.SymFormatSuper,
	ast.FormatSub:    sexpr.SymFormatSub,
	ast.FormatQuote:  sexpr.SymFormatQuote,
	ast.FormatSpan:   sexpr.SymFormatSpan,
}

var mapLiteralKindS = map[ast.LiteralKind]*sexpr.Symbol{
	ast.LiteralZettel:  sexpr.SymLiteralZettel,
	ast.LiteralProg:    sexpr.SymLiteralProg,
	ast.LiteralInput:   sexpr.SymLiteralInput,
	ast.LiteralOutput:  sexpr.SymLiteralOutput,
	ast.LiteralComment: sexpr.SymLiteralComment,
	ast.LiteralHTML:    sexpr.SymLiteralHTML,
	ast.LiteralMath:    sexpr.SymLiteralMath,
}

func (t *transformer) getBlockSlice(bs *ast.BlockSlice) *sexpr.List {
	lstVals := make([]sexpr.Value, len(*bs))
	for i, n := range *bs {
		lstVals[i] = t.getSexpr(n)
	}
	return sexpr.NewList(lstVals...)
}
func (t *transformer) getInlineSlice(is ast.InlineSlice) *sexpr.List {
	lstVals := make([]sexpr.Value, len(is))
	for i, n := range is {
		lstVals[i] = t.getSexpr(n)
	}
	return sexpr.NewList(lstVals...)
}

func getAttributes(a zjson.Attributes) *sexpr.List {
	if a.IsEmpty() {
		return sexpr.NewList()
	}
	keys := a.Keys()
	lstVals := make([]sexpr.Value, 0, len(keys))
	for _, k := range keys {
		lstVals = append(lstVals, sexpr.NewList(sexpr.NewString(k), sexpr.NewString(a[k])))
	}
	return sexpr.NewList(lstVals...)
}

var mapRefStateS = map[ast.RefState]*sexpr.Symbol{
	ast.RefStateInvalid:  sexpr.SymRefStateInvalid,
	ast.RefStateZettel:   sexpr.SymRefStateZettel,
	ast.RefStateSelf:     sexpr.SymRefStateSelf,
	ast.RefStateFound:    sexpr.SymRefStateFound,
	ast.RefStateBroken:   sexpr.SymRefStateBroken,
	ast.RefStateHosted:   sexpr.SymRefStateHosted,
	ast.RefStateBased:    sexpr.SymRefStateBased,
	ast.RefStateExternal: sexpr.SymRefStateExternal,
}

func getReference(ref *ast.Reference) *sexpr.List {
	return sexpr.NewList(mapGetS(mapRefStateS, ref.State), sexpr.NewString(ref.Value))
}

var mapMetaTypeS = map[*meta.DescriptionType]*sexpr.Symbol{
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

func (t *transformer) getMeta(m *meta.Meta, evalMeta encoder.EvalMetaFunc) *sexpr.List {
	pairs := m.ComputedPairs()
	lstVals := make([]sexpr.Value, 0, len(pairs))
	for _, p := range pairs {
		key := p.Key
		ty := m.Type(key)
		symType := mapGetS(mapMetaTypeS, ty)
		symKey := sexpr.GetSymbol(key)
		var val sexpr.Value
		if ty.IsSet {
			setList := meta.ListFromValue(p.Value)
			setVals := make([]sexpr.Value, len(setList))
			for i, val := range setList {
				setVals[i] = sexpr.NewString(val)
			}
			val = sexpr.NewList(setVals...)
		} else if ty == meta.TypeZettelmarkup {
			is := evalMeta(p.Value)
			val = t.getSexpr(&is)
		} else {
			val = sexpr.NewString(p.Value)
		}
		lstVals = append(lstVals, sexpr.NewList(symType, symKey, val))
	}
	return sexpr.NewList(lstVals...)
}

func mapGetS[T comparable](m map[T]*sexpr.Symbol, k T) *sexpr.Symbol {
	if result, found := m[k]; found {
		return result
	}
	log.Println("MISS", k, m)
	return sexpr.GetSymbol(fmt.Sprintf("**%v:not-found**", k))
}

func getBase64String(data []byte) *sexpr.String {
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	_, err := encoder.Write(data)
	if err == nil {
		err = encoder.Close()
	}
	if err == nil {
		return sexpr.NewString(buf.String())
	}
	return nil
}
