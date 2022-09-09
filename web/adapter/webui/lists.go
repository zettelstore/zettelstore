//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package webui

import (
	"bytes"
	"net/http"
	"net/url"
	"sort"
	"strconv"

	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/evaluator"
	"zettelstore.de/z/search"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/server"
)

// MakeListHTMLMetaHandler creates a HTTP handler for rendering the list of
// zettel as HTML.
func (wui *WebUI) MakeListHTMLMetaHandler(
	listMeta usecase.ListMeta,
	listRole usecase.ListRoles,
	listTags usecase.ListTags,
	evaluate *usecase.Evaluate,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		switch query.Get("_l") {
		case "t":
			wui.renderTagsList(w, r, listTags)
		default:
			wui.renderZettelList(w, r, listMeta, evaluate)
		}
	}
}

func (wui *WebUI) renderZettelList(
	w http.ResponseWriter, r *http.Request,
	listMeta usecase.ListMeta, evaluate *usecase.Evaluate,
) {
	sea := adapter.GetSearch(r.URL.Query())
	ctx := r.Context()
	if !sea.EnrichNeeded() {
		ctx = box.NoEnrichContext(ctx)
	}
	metaList, err := listMeta.Run(ctx, sea)
	if err != nil {
		wui.reportError(ctx, w, err)
		return
	}
	bns := evaluate.RunBlockNode(ctx, evaluator.ActionSearch(sea, metaList))
	enc := wui.getSimpleHTMLEncoder()
	htmlContent, err := enc.BlocksString(&bns)
	if err != nil {
		wui.reportError(ctx, w, err)
		return
	}
	user := server.GetUser(ctx)
	var base baseData
	wui.makeBaseData(ctx, wui.rtConfig.Get(ctx, nil, api.KeyLang), wui.rtConfig.GetSiteName(), "", user, &base)
	wui.renderTemplate(ctx, w, id.ListTemplateZid, &base, struct {
		Title          string
		SearchURL      string
		SearchValue    string
		QueryKeySearch string
		Content        string
	}{
		Title:          wui.listTitleSearch(sea),
		SearchURL:      base.SearchURL,
		SearchValue:    sea.String(),
		QueryKeySearch: base.QueryKeySearch,
		Content:        htmlContent,
	})
}

type countInfo struct {
	Count string
	URL   string
}

type tagInfo struct {
	Name   string
	URL    string
	iCount int
	Count  string
	Size   string
}

const fontSizes = 6 // Must be the number of CSS classes zs-font-size-* in base.css

func (wui *WebUI) renderTagsList(w http.ResponseWriter, r *http.Request, listTags usecase.ListTags) {
	ctx := r.Context()
	iMinCount, err := strconv.Atoi(r.URL.Query().Get("min"))
	if err != nil || iMinCount < 0 {
		iMinCount = 0
	}
	tagData, err := listTags.Run(ctx, iMinCount)
	if err != nil {
		wui.reportError(ctx, w, err)
		return
	}

	user := server.GetUser(ctx)
	tagsList := make([]tagInfo, 0, len(tagData))
	countMap := make(map[int]int)
	baseTagListURL := wui.NewURLBuilder('h')
	for tag, ml := range tagData {
		count := len(ml)
		countMap[count]++
		tagsList = append(
			tagsList,
			tagInfo{tag, baseTagListURL.AppendSearch(api.KeyAllTags + api.SearchOperatorHas + tag).String(), count, "", ""})
		baseTagListURL.ClearQuery()
	}
	sort.Slice(tagsList, func(i, j int) bool { return tagsList[i].Name < tagsList[j].Name })

	countList := make([]int, 0, len(countMap))
	for count := range countMap {
		countList = append(countList, count)
	}
	sort.Ints(countList)
	for pos, count := range countList {
		countMap[count] = (pos * fontSizes) / len(countList)
	}
	for i := 0; i < len(tagsList); i++ {
		count := tagsList[i].iCount
		tagsList[i].Count = strconv.Itoa(count)
		tagsList[i].Size = strconv.Itoa(countMap[count])
	}

	var base baseData
	wui.makeBaseData(ctx, wui.rtConfig.Get(ctx, nil, api.KeyLang), wui.rtConfig.GetSiteName(), "", user, &base)
	minCounts := make([]countInfo, 0, len(countList))
	for _, c := range countList {
		sCount := strconv.Itoa(c)
		minCounts = append(minCounts, countInfo{sCount, base.ListTagsURL + "&min=" + sCount})
	}

	wui.renderTemplate(ctx, w, id.TagsTemplateZid, &base, struct {
		ListTagsURL string
		MinCounts   []countInfo
		Tags        []tagInfo
	}{
		ListTagsURL: base.ListTagsURL,
		MinCounts:   minCounts,
		Tags:        tagsList,
	})
}

// MakeZettelContextHandler creates a new HTTP handler for the use case "zettel context".
func (wui *WebUI) MakeZettelContextHandler(getContext usecase.ZettelContext, evaluate *usecase.Evaluate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			wui.reportError(ctx, w, box.ErrNotFound)
			return
		}
		q := r.URL.Query()
		dir := adapter.GetZCDirection(q.Get(api.QueryKeyDir))
		depth := getIntParameter(q, api.QueryKeyDepth, 5)
		limit := getIntParameter(q, api.QueryKeyLimit, 200)
		metaList, err := getContext.Run(ctx, zid, dir, depth, limit)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		bns := evaluate.RunBlockNode(ctx, evaluator.ActionSearch(nil, metaList))
		enc := wui.getSimpleHTMLEncoder()
		htmlContent, err := enc.BlocksString(&bns)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		apiZid := api.ZettelID(zid.String())
		depths := []string{"2", "3", "4", "5", "6", "7", "8", "9", "10"}
		depthLinks := make([]simpleLink, len(depths))
		depthURL := wui.NewURLBuilder('k').SetZid(apiZid)
		for i, depth := range depths {
			depthURL.ClearQuery()
			switch dir {
			case usecase.ZettelContextBackward:
				depthURL.AppendQuery(api.QueryKeyDir, api.DirBackward)
			case usecase.ZettelContextForward:
				depthURL.AppendQuery(api.QueryKeyDir, api.DirForward)
			}
			depthURL.AppendQuery(api.QueryKeyDepth, depth)
			depthLinks[i].Text = depth
			depthLinks[i].URL = depthURL.String()
		}
		var base baseData
		user := server.GetUser(ctx)
		wui.makeBaseData(ctx, wui.rtConfig.Get(ctx, nil, api.KeyLang), wui.rtConfig.GetSiteName(), "", user, &base)
		wui.renderTemplate(ctx, w, id.ContextTemplateZid, &base, struct {
			Title   string
			InfoURL string
			Depths  []simpleLink
			Content string
		}{
			Title:   "Zettel Context",
			InfoURL: wui.NewURLBuilder('i').SetZid(apiZid).String(),
			Depths:  depthLinks,
			Content: htmlContent,
		})
	}
}

func getIntParameter(q url.Values, key string, minValue int) int {
	val, ok := adapter.GetInteger(q, key)
	if !ok || val < 0 {
		return minValue
	}
	return val
}

func (wui *WebUI) listTitleSearch(s *search.Search) string {
	if s == nil {
		return wui.rtConfig.GetSiteName()
	}
	var buf bytes.Buffer
	s.PrintHuman(&buf)
	return buf.String()
}
