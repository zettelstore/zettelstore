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
	"log"
	"sort"
	"strconv"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/c/attrs"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoding/rss"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/query"
)

// ActionSearch transforms a list of metadata according to search commands into a AST nested list.
func ActionSearch(q *query.Query, ml []*meta.Meta, rtConfig config.Config) ast.BlockNode {
	ap := actionPara{
		q:     q,
		ml:    ml,
		kind:  ast.NestedListUnordered,
		min:   -1,
		max:   -1,
		title: rtConfig.GetSiteName(),
	}
	if actions := q.Actions(); len(actions) > 0 {
		acts := make([]string, 0, len(actions))
		for i, act := range actions {
			if strings.HasPrefix(act, "N") {
				ap.kind = ast.NestedListOrdered
				continue
			}
			if strings.HasPrefix(act, "MIN") {
				if num, err := strconv.Atoi(act[3:]); err == nil && num > 0 {
					ap.min = num
					continue
				}
			}
			if strings.HasPrefix(act, "MAX") {
				if num, err := strconv.Atoi(act[3:]); err == nil && num > 0 {
					ap.max = num
					continue
				}
			}
			if act == "TITLE" && i+1 < len(actions) {
				ap.title = strings.Join(actions[i+1:], " ")
				break
			}
			acts = append(acts, act)
		}
		for _, act := range acts {
			if act == "RSS" {
				return ap.createBlockNodeRSS()
			}
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
	q     *query.Query
	ml    []*meta.Meta
	kind  ast.NestedListKind
	min   int
	max   int
	title string
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
	if min, max := ap.min, ap.max; min > 0 || max > 0 {
		if min < 0 {
			min = ccs[len(ccs)-1].Count
		}
		if max < 0 {
			max = ccs[0].Count
		}
		if ccs[len(ccs)-1].Count < min || max < ccs[0].Count {
			temp := make(meta.CountedCategories, 0, len(ccs))
			for _, cat := range ccs {
				if min <= cat.Count && cat.Count <= max {
					temp = append(temp, cat)
				}
			}
			ccs = temp
		}
	}
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

	sea := ap.q.Clone()
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
	if len(countList) <= fontSizes {
		// If we have less different counts, center them inside the fsAttrs vector.
		curSize := (fontSizes - len(countList)) / 2
		for _, count := range countList {
			result[count] = fsAttrs[curSize]
			curSize++
		}
		return result
	}

	// Idea: the number of occurences for a specific count is substracted from a budget.
	budget := len(ccs) / (fontSizes - 1)
	curBudget := budget
	curSize := 0
	for _, count := range countList {
		result[count] = fsAttrs[curSize]
		curBudget -= countMap[count]
		for curBudget <= 0 {
			curBudget += budget
			curSize++
			if curSize >= fontSizes {
				curSize = fontSizes - 1
			}
		}
	}
	return result
}

func (ap *actionPara) createBlockNodeRSS() ast.BlockNode {
	config := rss.Configuration{
		Title: ap.title,
		NewURLBuilderAbs: func(key byte) *api.URLBuilder {
			return api.NewURLBuilder("http://127.0.0.1:23123/", key)
		},
		// GetBaseURL:   ap.config.GetBaseURL,
		// GetZettelURL: func(zid id.Zid) string { return ap.config.GetBaseURL() + zid.String() },
		// Encrypt:      ap.config.Encrypt,
	}
	data, err := config.Marshal(ap.ml)
	if err != nil {
		log.Println("ERRR", err)
		return nil
	}
	return &ast.VerbatimNode{
		Kind:    ast.VerbatimProg,
		Attrs:   attrs.Attributes{"lang": "xml"},
		Content: data,
	}
}
