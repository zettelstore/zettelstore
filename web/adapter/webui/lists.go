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
	"context"
	"net/http"
	"net/url"
	"sort"
	"strconv"

	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/search"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
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
		case "r":
			wui.renderRolesList(w, r, listRole)
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
	query := r.URL.Query()
	s := adapter.GetSearch(query)
	ctx := r.Context()
	title := wui.listTitleSearch(s)

	if !s.EnrichNeeded() {
		ctx = box.NoEnrichContext(ctx)
	}
	metaList, err := listMeta.Run(ctx, s)
	if err != nil {
		wui.reportError(ctx, w, err)
		return
	}
	user := wui.getUser(ctx)
	metas := wui.buildHTMLMetaList(ctx, metaList, evaluate)
	var base baseData
	wui.makeBaseData(ctx, wui.rtConfig.GetDefaultLang(), wui.rtConfig.GetSiteName(), "", user, &base)
	wui.renderTemplate(ctx, w, id.ListTemplateZid, &base, struct {
		Title string
		Metas []simpleLink
	}{
		Title: title,
		Metas: metas,
	})
}

type roleInfo struct {
	Text string
	URL  string
}

func (wui *WebUI) renderRolesList(w http.ResponseWriter, r *http.Request, listRole usecase.ListRoles) {
	ctx := r.Context()
	roleArrangement, err := listRole.Run(ctx)
	if err != nil {
		wui.reportError(ctx, w, err)
		return
	}
	roleList := roleArrangement.Counted()
	roleList.SortByName()

	roleInfos := make([]roleInfo, len(roleList))
	for i, role := range roleList {
		roleInfos[i] = roleInfo{role.Name, wui.NewURLBuilder('h').AppendQuery("role", role.Name).String()}
	}

	user := wui.getUser(ctx)
	var base baseData
	wui.makeBaseData(ctx, wui.rtConfig.GetDefaultLang(), wui.rtConfig.GetSiteName(), "", user, &base)
	wui.renderTemplate(ctx, w, id.RolesTemplateZid, &base, struct {
		Roles []roleInfo
	}{
		Roles: roleInfos,
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

	user := wui.getUser(ctx)
	tagsList := make([]tagInfo, 0, len(tagData))
	countMap := make(map[int]int)
	baseTagListURL := wui.NewURLBuilder('h')
	for tag, ml := range tagData {
		count := len(ml)
		countMap[count]++
		tagsList = append(
			tagsList,
			tagInfo{tag, baseTagListURL.AppendQuery(api.KeyAllTags, tag).String(), count, "", ""})
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
	wui.makeBaseData(ctx, wui.rtConfig.GetDefaultLang(), wui.rtConfig.GetSiteName(), "", user, &base)
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
		apiZid := api.ZettelID(zid.String())
		metaLinks := wui.buildHTMLMetaList(ctx, metaList, evaluate)
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
		user := wui.getUser(ctx)
		wui.makeBaseData(ctx, wui.rtConfig.GetDefaultLang(), wui.rtConfig.GetSiteName(), "", user, &base)
		wui.renderTemplate(ctx, w, id.ContextTemplateZid, &base, struct {
			Title   string
			InfoURL string
			Depths  []simpleLink
			Start   simpleLink
			Metas   []simpleLink
		}{
			Title:   "Zettel Context",
			InfoURL: wui.NewURLBuilder('i').SetZid(apiZid).String(),
			Depths:  depthLinks,
			Start:   metaLinks[0],
			Metas:   metaLinks[1:],
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
	s.Print(&buf)
	return buf.String()
}

// buildHTMLMetaList builds a zettel list based on a meta list for HTML rendering.
func (wui *WebUI) buildHTMLMetaList(ctx context.Context, metaList []*meta.Meta, evaluate *usecase.Evaluate) []simpleLink {
	metas := make([]simpleLink, 0, len(metaList))
	encHTML := wui.getSimpleHTMLEncoder()
	for _, m := range metaList {
		metas = append(metas, simpleLink{
			Text: wui.encodeTitleAsHTML(ctx, m, evaluate, encHTML, true),
			URL:  wui.NewURLBuilder('h').SetZid(api.ZettelID(m.Zid.String())).String(),
		})
	}
	return metas
}
