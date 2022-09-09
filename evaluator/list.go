//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package evaluator

import (
	"bytes"
	"sort"
	"strconv"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/c/attrs"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/search"
)

// ActionSearch transforms a list of metadata according to search commands into a AST nested list.
func ActionSearch(sea *search.Search, ml []*meta.Meta) ast.BlockNode {
	ap := actionPara{
		sea:  sea,
		ml:   ml,
		kind: ast.NestedListUnordered,
	}
	if actions := sea.Actions(); len(actions) > 0 {
		acts := make([]string, 0, len(actions))
		for _, act := range actions {
			if strings.HasPrefix(act, "N") {
				ap.kind = ast.NestedListOrdered
				continue
			}
			acts = append(acts, act)
		}
		for _, act := range acts {
			key := strings.ToLower(act)
			switch meta.Type(key) {
			case meta.TypeWord:
				return ap.createBlockNodeWord(key)
			case meta.TypeTagSet:
				return ap.createBlockNodeTagSet(key)
			}
		}
	}
	return ap.createBlockNodeMeta()
}

type actionPara struct {
	sea  *search.Search
	ml   []*meta.Meta
	kind ast.NestedListKind
}

func (ap *actionPara) createBlockNodeWord(key string) ast.BlockNode {
	var buf bytes.Buffer
	ccs, bufLen := ap.prepareCatAction(key, &buf)
	if len(ccs) == 0 {
		return nil
	}
	items := make([]ast.ItemSlice, 0, len(ccs))
	ccs.SortByName()
	for _, cat := range ccs {
		buf.WriteString(cat.Name)
		items = append(items, ast.ItemSlice{ast.CreateParaNode(&ast.LinkNode{
			Attrs:   nil,
			Ref:     ast.ParseReference(buf.String()),
			Inlines: ast.InlineSlice{&ast.TextNode{Text: cat.Name}},
		})})
		buf.Truncate(bufLen)
	}
	return &ast.NestedListNode{
		Kind:  ap.kind,
		Items: items,
		Attrs: nil,
	}
}

func (ap *actionPara) createBlockNodeTagSet(key string) ast.BlockNode {
	var buf bytes.Buffer
	ccs, bufLen := ap.prepareCatAction(key, &buf)
	if len(ccs) == 0 {
		return nil
	}
	ccs.SortByCount()
	countMap := ap.calcFontSizes(ccs)

	para := make(ast.InlineSlice, 0, len(ccs))
	ccs.SortByName()
	for i, cat := range ccs {
		if i > 0 {
			para = append(para, &ast.SpaceNode{
				Lexeme: " ",
			})
		}
		buf.WriteString(cat.Name)
		para = append(para,
			&ast.LinkNode{
				Attrs: countMap[cat.Count],
				Ref:   ast.ParseReference(buf.String()),
				Inlines: ast.InlineSlice{
					&ast.TextNode{Text: cat.Name},
				},
			},
			&ast.FormatNode{
				Kind:    ast.FormatSuper,
				Attrs:   nil,
				Inlines: ast.InlineSlice{&ast.TextNode{Text: strconv.Itoa(cat.Count)}},
			},
		)
		buf.Truncate(bufLen)
	}
	return &ast.ParaNode{
		Inlines: para,
	}
}

func (ap *actionPara) createBlockNodeMeta() ast.BlockNode {
	if len(ap.ml) == 0 {
		return nil
	}
	items := make([]ast.ItemSlice, 0, len(ap.ml))
	for _, m := range ap.ml {
		zid := m.Zid.String()
		title, found := m.Get(api.KeyTitle)
		if !found {
			title = zid
		}
		items = append(items, ast.ItemSlice{ast.CreateParaNode(&ast.LinkNode{
			Attrs:   nil,
			Ref:     ast.ParseReference(zid),
			Inlines: parser.ParseMetadataNoLink(title),
		})})
	}
	return &ast.NestedListNode{
		Kind:  ap.kind,
		Items: items,
		Attrs: nil,
	}
}

func (ap *actionPara) prepareCatAction(key string, buf *bytes.Buffer) (meta.CountedCategories, int) {
	if len(ap.ml) == 0 {
		return nil, 0
	}
	ccs := meta.CreateArrangement(ap.ml, key).Counted()
	if len(ccs) == 0 {
		return nil, 0
	}

	sea := ap.sea.Clone()
	sea.RemoveActions()
	buf.WriteString(ast.SearchPrefix)
	sea.Print(buf)
	if buf.Len() > len(ast.SearchPrefix) {
		buf.WriteByte(' ')
	}
	buf.WriteString(key)
	buf.WriteByte(':')
	bufLen := buf.Len()

	return ccs, bufLen
}

const fontSizes = 6 // Must be the number of CSS classes zs-font-size-* in base.css

func (*actionPara) calcFontSizes(ccs meta.CountedCategories) map[int]attrs.Attributes {
	var fsAttrs [fontSizes]attrs.Attributes
	var a attrs.Attributes
	for i := 0; i < fontSizes; i++ {
		fsAttrs[i] = a.AddClass("zs-font-size-" + strconv.Itoa(i))
	}

	countMap := make(map[int]int, len(ccs))
	for _, cat := range ccs {
		countMap[cat.Count]++
	}

	countList := make([]int, 0, len(countMap))
	for count := range countMap {
		countList = append(countList, count)
	}
	sort.Ints(countList)

	result := make(map[int]attrs.Attributes, len(countList))
	for pos, count := range countList {
		result[count] = fsAttrs[(pos*fontSizes)/len(countList)]
	}
	return result
}
