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

	"codeberg.org/t73fde/sxpf"
	"zettelstore.de/c/api"
	"zettelstore.de/c/attrs"
	"zettelstore.de/c/sexpr"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
)

// GetSexpr returns the given node as a s-expression.
func GetSexpr(node ast.Node) *sxpf.Pair {
	t := transformer{}
	return t.getSexpr(node)
}

type transformer struct {
	inVerse bool
}

func (t *transformer) getSexpr(node ast.Node) *sxpf.Pair {
	switch n := node.(type) {
	case *ast.BlockSlice:
		return t.getBlockSlice(n)
	case *ast.InlineSlice:
		return t.getInlineSlice(*n)
	case *ast.ParaNode:
		return sxpf.NewPair(sexpr.SymPara, t.getInlineSlice(n.Inlines))
	case *ast.VerbatimNode:
		return sxpf.NewPairFromValues(
			mapGetS(mapVerbatimKindS, n.Kind),
			getAttributes(n.Attrs),
			sxpf.NewString(string(n.Content)),
		)
	case *ast.RegionNode:
		return t.getRegion(n)
	case *ast.HeadingNode:
		return sxpf.NewPair(
			sexpr.SymHeading,
			sxpf.NewPair(
				sxpf.NewInteger(int64(n.Level)),
				sxpf.NewPair(
					getAttributes(n.Attrs),
					sxpf.NewPair(
						sxpf.NewString(n.Slug),
						sxpf.NewPair(
							sxpf.NewString(n.Fragment),
							t.getInlineSlice(n.Inlines),
						),
					),
				),
			),
		)
	case *ast.HRuleNode:
		return sxpf.NewPairFromValues(sexpr.SymThematic, getAttributes(n.Attrs))
	case *ast.NestedListNode:
		return t.getNestedList(n)
	case *ast.DescriptionListNode:
		return t.getDescriptionList(n)
	case *ast.TableNode:
		return t.getTable(n)
	case *ast.TranscludeNode:
		return sxpf.NewPairFromValues(sexpr.SymTransclude, getReference(n.Ref))
	case *ast.BLOBNode:
		return getBLOB(n)
	case *ast.TextNode:
		return sxpf.NewPairFromValues(sexpr.SymText, sxpf.NewString(n.Text))
	case *ast.TagNode:
		return sxpf.NewPairFromValues(sexpr.SymTag, sxpf.NewString(n.Tag))
	case *ast.SpaceNode:
		if t.inVerse {
			return sxpf.NewPairFromValues(sexpr.SymSpace, sxpf.NewString(n.Lexeme))
		}
		return sxpf.NewPairFromValues(sexpr.SymSpace)
	case *ast.BreakNode:
		if n.Hard {
			return sxpf.NewPairFromValues(sexpr.SymHard)
		} else {
			return sxpf.NewPairFromValues(sexpr.SymSoft)
		}
	case *ast.LinkNode:
		return t.getLink(n)
	case *ast.EmbedRefNode:
		return sxpf.NewPair(
			sexpr.SymEmbed,
			sxpf.NewPair(
				getAttributes(n.Attrs),
				sxpf.NewPair(
					getReference(n.Ref),
					sxpf.NewPair(
						sxpf.NewString(n.Syntax),
						t.getInlineSlice(n.Inlines),
					),
				),
			),
		)
	case *ast.EmbedBLOBNode:
		return t.getEmbedBLOB(n)
	case *ast.CiteNode:
		return sxpf.NewPair(
			sexpr.SymCite,
			sxpf.NewPair(
				getAttributes(n.Attrs),
				sxpf.NewPair(
					sxpf.NewString(n.Key),
					t.getInlineSlice(n.Inlines),
				),
			),
		)
	case *ast.FootnoteNode:
		return sxpf.NewPair(
			sexpr.SymFootnote,
			sxpf.NewPair(
				getAttributes(n.Attrs),
				t.getInlineSlice(n.Inlines),
			),
		)
	case *ast.MarkNode:
		return sxpf.NewPair(
			sexpr.SymMark,
			sxpf.NewPair(
				sxpf.NewString(n.Mark),
				sxpf.NewPair(
					sxpf.NewString(n.Slug),
					sxpf.NewPair(
						sxpf.NewString(n.Fragment),
						t.getInlineSlice(n.Inlines),
					),
				),
			),
		)
	case *ast.FormatNode:
		return sxpf.NewPair(
			mapGetS(mapFormatKindS, n.Kind),
			sxpf.NewPair(
				getAttributes(n.Attrs),
				t.getInlineSlice(n.Inlines),
			),
		)
	case *ast.LiteralNode:
		return sxpf.NewPairFromValues(
			mapGetS(mapLiteralKindS, n.Kind),
			getAttributes(n.Attrs),
			sxpf.NewString(string(n.Content)),
		)
	}
	log.Printf("SEXPR %T %v\n", node, node)
	return sxpf.NewPairFromValues(sexpr.SymUnknown, sxpf.NewString(fmt.Sprintf("%T %v", node, node)))
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

func (t *transformer) getRegion(rn *ast.RegionNode) *sxpf.Pair {
	saveInVerse := t.inVerse
	if rn.Kind == ast.RegionVerse {
		t.inVerse = true
	}
	symBlocks := t.getSexpr(&rn.Blocks)
	t.inVerse = saveInVerse
	return sxpf.NewPairFromValues(
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

func (t *transformer) getNestedList(ln *ast.NestedListNode) *sxpf.Pair {
	nlistVals := make([]sxpf.Value, len(ln.Items)+1)
	nlistVals[0] = mapGetS(mapNestedListKindS, ln.Kind)
	isCompact := isCompactList(ln.Items)
	for i, item := range ln.Items {
		if isCompact && len(item) > 0 {
			paragraph := t.getSexpr(item[0])
			nlistVals[i+1] = paragraph.GetTail()
			continue
		}
		itemVals := make([]sxpf.Value, len(item))
		for j, in := range item {
			itemVals[j] = t.getSexpr(in)
		}
		nlistVals[i+1] = sxpf.NewPairFromValues(itemVals...)
	}
	return sxpf.NewPairFromValues(nlistVals...)
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

func (t *transformer) getDescriptionList(dn *ast.DescriptionListNode) *sxpf.Pair {
	dlVals := make([]sxpf.Value, 2*len(dn.Descriptions)+1)
	dlVals[0] = sexpr.SymDescription
	for i, def := range dn.Descriptions {
		dlVals[2*i+1] = t.getInlineSlice(def.Term)
		descVals := make([]sxpf.Value, len(def.Descriptions))
		for j, b := range def.Descriptions {
			if len(b) == 1 {
				descVals[j] = t.getSexpr(b[0]).GetTail()
				continue
			}
			dVal := make([]sxpf.Value, len(b))
			for k, dn := range b {
				dVal[k] = t.getSexpr(dn)
			}
			descVals[j] = sxpf.NewPairFromValues(dVal...)
		}
		dlVals[2*i+2] = sxpf.NewPairFromValues(descVals...)
	}
	return sxpf.NewPairFromValues(dlVals...)
}

func (t *transformer) getTable(tn *ast.TableNode) *sxpf.Pair {
	tVals := make([]sxpf.Value, len(tn.Rows)+2)
	tVals[0] = sexpr.SymTable
	tVals[1] = t.getRow(tn.Header)
	for i, row := range tn.Rows {
		tVals[i+2] = t.getRow(row)
	}
	return sxpf.NewPairFromValues(tVals...)
}
func (t *transformer) getRow(row ast.TableRow) *sxpf.Pair {
	rVals := make([]sxpf.Value, len(row))
	for i, cell := range row {
		rVals[i] = t.getCell(cell)
	}
	return sxpf.NewPairFromValues(rVals...)
}

var alignmentSymbolS = map[ast.Alignment]*sxpf.Symbol{
	ast.AlignDefault: sexpr.SymCell,
	ast.AlignLeft:    sexpr.SymCellLeft,
	ast.AlignCenter:  sexpr.SymCellCenter,
	ast.AlignRight:   sexpr.SymCellRight,
}

func (t *transformer) getCell(cell *ast.TableCell) *sxpf.Pair {
	return sxpf.NewPair(mapGetS(alignmentSymbolS, cell.Align), t.getInlineSlice(cell.Inlines))
}

func getBLOB(bn *ast.BLOBNode) *sxpf.Pair {
	var lastValue sxpf.Value
	if bn.Syntax == api.ValueSyntaxSVG {
		lastValue = sxpf.NewString(string(bn.Blob))
	} else {
		lastValue = getBase64String(bn.Blob)
	}
	return sxpf.NewPairFromValues(
		sexpr.SymBLOB,
		sxpf.NewString(bn.Title),
		sxpf.NewString(bn.Syntax),
		lastValue,
	)
}

var mapRefStateLink = map[ast.RefState]*sxpf.Symbol{
	ast.RefStateInvalid:  sexpr.SymLinkInvalid,
	ast.RefStateZettel:   sexpr.SymLinkZettel,
	ast.RefStateSelf:     sexpr.SymLinkSelf,
	ast.RefStateFound:    sexpr.SymLinkFound,
	ast.RefStateBroken:   sexpr.SymLinkBroken,
	ast.RefStateHosted:   sexpr.SymLinkHosted,
	ast.RefStateBased:    sexpr.SymLinkBased,
	ast.RefStateSearch:   sexpr.SymLinkSearch,
	ast.RefStateExternal: sexpr.SymLinkExternal,
}

func (t *transformer) getLink(ln *ast.LinkNode) *sxpf.Pair {
	return sxpf.NewPair(
		mapGetS(mapRefStateLink, ln.Ref.State),
		sxpf.NewPair(
			getAttributes(ln.Attrs),
			sxpf.NewPair(
				sxpf.NewString(ln.Ref.Value),
				t.getInlineSlice(ln.Inlines),
			),
		),
	)
}

func (t *transformer) getEmbedBLOB(en *ast.EmbedBLOBNode) *sxpf.Pair {
	tail := t.getInlineSlice(en.Inlines)
	if en.Syntax == api.ValueSyntaxSVG {
		tail = sxpf.NewPair(sxpf.NewString(string(en.Blob)), tail)
	} else {
		tail = sxpf.NewPair(getBase64String(en.Blob), tail)
	}
	return sxpf.NewPair(
		sexpr.SymEmbedBLOB,
		sxpf.NewPair(
			getAttributes(en.Attrs),
			sxpf.NewPair(
				sxpf.NewString(en.Syntax),
				tail,
			),
		),
	)
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

func (t *transformer) getBlockSlice(bs *ast.BlockSlice) *sxpf.Pair {
	lstVals := make([]sxpf.Value, len(*bs))
	for i, n := range *bs {
		lstVals[i] = t.getSexpr(n)
	}
	return sxpf.NewPairFromSlice(lstVals)
}
func (t *transformer) getInlineSlice(is ast.InlineSlice) *sxpf.Pair {
	lstVals := make([]sxpf.Value, len(is))
	for i, n := range is {
		lstVals[i] = t.getSexpr(n)
	}
	return sxpf.NewPairFromSlice(lstVals)
}

func getAttributes(a attrs.Attributes) sxpf.Value {
	if a.IsEmpty() {
		return sxpf.Nil()
	}
	keys := a.Keys()
	lstVals := make([]sxpf.Value, 0, len(keys))
	for _, k := range keys {
		lstVals = append(lstVals, sxpf.NewPair(sxpf.NewString(k), sxpf.NewPair(sxpf.NewString(a[k]), nil)))
	}
	return sxpf.NewPairFromSlice(lstVals)
}

var mapRefStateS = map[ast.RefState]*sxpf.Symbol{
	ast.RefStateInvalid:  sexpr.SymRefStateInvalid,
	ast.RefStateZettel:   sexpr.SymRefStateZettel,
	ast.RefStateSelf:     sexpr.SymRefStateSelf,
	ast.RefStateFound:    sexpr.SymRefStateFound,
	ast.RefStateBroken:   sexpr.SymRefStateBroken,
	ast.RefStateHosted:   sexpr.SymRefStateHosted,
	ast.RefStateBased:    sexpr.SymRefStateBased,
	ast.RefStateSearch:   sexpr.SymRefStateSearch,
	ast.RefStateExternal: sexpr.SymRefStateExternal,
}

func getReference(ref *ast.Reference) *sxpf.Pair {
	return sxpf.NewPair(
		mapGetS(mapRefStateS, ref.State),
		sxpf.NewPair(
			sxpf.NewString(ref.Value),
			sxpf.Nil()))
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

func GetMeta(m *meta.Meta, evalMeta encoder.EvalMetaFunc) *sxpf.Pair {
	pairs := m.ComputedPairs()
	lstVals := make([]sxpf.Value, 0, len(pairs))
	for _, p := range pairs {
		key := p.Key
		ty := m.Type(key)
		symType := mapGetS(mapMetaTypeS, ty)
		strKey := sxpf.NewString(key)
		var val sxpf.Value
		if ty.IsSet {
			setList := meta.ListFromValue(p.Value)
			setVals := make([]sxpf.Value, len(setList))
			for i, val := range setList {
				setVals[i] = sxpf.NewString(val)
			}
			val = sxpf.NewPairFromSlice(setVals)
		} else if ty == meta.TypeZettelmarkup {
			is := evalMeta(p.Value)
			t := transformer{}
			val = t.getSexpr(&is)
		} else {
			val = sxpf.NewString(p.Value)
		}
		lstVals = append(lstVals, sxpf.NewPair(symType, sxpf.NewPair(strKey, sxpf.NewPair(val, nil))))
	}
	return sxpf.NewPairFromSlice(lstVals)
}

func mapGetS[T comparable](m map[T]*sxpf.Symbol, k T) *sxpf.Symbol {
	if result, found := m[k]; found {
		return result
	}
	log.Println("MISS", k, m)
	return sexpr.Smk.MakeSymbol(fmt.Sprintf("**%v:NOT-FOUND**", k))
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
