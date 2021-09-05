//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package webui provides web-UI handlers for web requests.
package webui

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"zettelstore.de/z/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/search"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeListHTMLMetaHandler creates a HTTP handler for rendering the list of
// zettel as HTML.
func (wui *WebUI) MakeListHTMLMetaHandler(
	listMeta usecase.ListMeta,
	listRole usecase.ListRole,
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
	title := wui.listTitleSearch("Select", s)
	wui.renderMetaList(
		ctx, w, title, s,
		func(s *search.Search) ([]*meta.Meta, error) {
			if !s.EnrichNeeded() {
				ctx = box.NoEnrichContext(ctx)
			}
			return listMeta.Run(ctx, s)
		},
		evaluate,
	)
}

type roleInfo struct {
	Text string
	URL  string
}

func (wui *WebUI) renderRolesList(w http.ResponseWriter, r *http.Request, listRole usecase.ListRole) {
	ctx := r.Context()
	roleList, err := listRole.Run(ctx)
	if err != nil {
		adapter.ReportUsecaseError(w, err)
		return
	}

	roleInfos := make([]roleInfo, 0, len(roleList))
	for _, role := range roleList {
		roleInfos = append(
			roleInfos,
			roleInfo{role, wui.NewURLBuilder('h').AppendQuery("role", role).String()})
	}

	user := wui.getUser(ctx)
	var base baseData
	wui.makeBaseData(ctx, wui.rtConfig.GetDefaultLang(), wui.rtConfig.GetSiteName(), user, &base)
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

var fontSizes = [...]int{75, 83, 100, 117, 150, 200}

func (wui *WebUI) renderTagsList(w http.ResponseWriter, r *http.Request, listTags usecase.ListTags) {
	ctx := r.Context()
	iMinCount, _ := strconv.Atoi(r.URL.Query().Get("min"))
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
			tagInfo{tag, baseTagListURL.AppendQuery("tags", tag).String(), count, "", ""})
		baseTagListURL.ClearQuery()
	}
	sort.Slice(tagsList, func(i, j int) bool { return tagsList[i].Name < tagsList[j].Name })

	countList := make([]int, 0, len(countMap))
	for count := range countMap {
		countList = append(countList, count)
	}
	sort.Ints(countList)
	for pos, count := range countList {
		countMap[count] = fontSizes[(pos*len(fontSizes))/len(countList)]
	}
	for i := 0; i < len(tagsList); i++ {
		count := tagsList[i].iCount
		tagsList[i].Count = strconv.Itoa(count)
		tagsList[i].Size = strconv.Itoa(countMap[count])
	}

	var base baseData
	wui.makeBaseData(ctx, wui.rtConfig.GetDefaultLang(), wui.rtConfig.GetSiteName(), user, &base)
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

// MakeSearchHandler creates a new HTTP handler for the use case "search".
func (wui *WebUI) MakeSearchHandler(ucSearch usecase.Search, evaluate *usecase.Evaluate) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		ctx := r.Context()
		s := adapter.GetSearch(query)
		if s == nil {
			redirectFound(w, r, wui.NewURLBuilder('h'))
			return
		}

		title := wui.listTitleSearch("Search", s)
		wui.renderMetaList(
			ctx, w, title, s, func(s *search.Search) ([]*meta.Meta, error) {
				if !s.EnrichNeeded() {
					ctx = box.NoEnrichContext(ctx)
				}
				return ucSearch.Run(ctx, s)
			},
			evaluate,
		)
	}
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
		metaLinks, err := wui.buildHTMLMetaList(ctx, metaList, evaluate)
		if err != nil {
			adapter.InternalServerError(w, "Build HTML meta list", err)
			return
		}

		depths := []string{"2", "3", "4", "5", "6", "7", "8", "9", "10"}
		depthLinks := make([]simpleLink, len(depths))
		depthURL := wui.NewURLBuilder('k').SetZid(zid)
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
		wui.makeBaseData(ctx, wui.rtConfig.GetDefaultLang(), wui.rtConfig.GetSiteName(), user, &base)
		wui.renderTemplate(ctx, w, id.ContextTemplateZid, &base, struct {
			Title   string
			InfoURL string
			Depths  []simpleLink
			Start   simpleLink
			Metas   []simpleLink
		}{
			Title:   "Zettel Context",
			InfoURL: wui.NewURLBuilder('i').SetZid(zid).String(),
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

func (wui *WebUI) renderMetaList(
	ctx context.Context,
	w http.ResponseWriter,
	title string,
	s *search.Search,
	ucMetaList func(sorter *search.Search) ([]*meta.Meta, error),
	evaluate *usecase.Evaluate,
) {

	metaList, err := ucMetaList(s)
	if err != nil {
		wui.reportError(ctx, w, err)
		return
	}
	user := wui.getUser(ctx)
	metas, err := wui.buildHTMLMetaList(ctx, metaList, evaluate)
	if err != nil {
		wui.reportError(ctx, w, err)
		return
	}
	var base baseData
	wui.makeBaseData(ctx, wui.rtConfig.GetDefaultLang(), wui.rtConfig.GetSiteName(), user, &base)
	wui.renderTemplate(ctx, w, id.ListTemplateZid, &base, struct {
		Title string
		Metas []simpleLink
	}{
		Title: title,
		Metas: metas,
	})
}

func (wui *WebUI) listTitleSearch(prefix string, s *search.Search) string {
	if s == nil {
		return wui.rtConfig.GetSiteName()
	}
	var sb strings.Builder
	sb.WriteString(prefix)
	if s != nil {
		sb.WriteString(": ")
		s.Print(&sb)
	}
	return sb.String()
}

// buildHTMLMetaList builds a zettel list based on a meta list for HTML rendering.
func (wui *WebUI) buildHTMLMetaList(
	ctx context.Context, metaList []*meta.Meta, evaluate *usecase.Evaluate,
) ([]simpleLink, error) {
	defaultLang := wui.rtConfig.GetDefaultLang()
	metas := make([]simpleLink, 0, len(metaList))
	for _, m := range metaList {
		var lang string
		if val, ok := m.Get(meta.KeyLang); ok {
			lang = val
		} else {
			lang = defaultLang
		}
		env := encoder.Environment{Lang: lang, Interactive: true}
		metas = append(metas, simpleLink{
			Text: wui.encodeTitleAsHTML(ctx, m, evaluate, nil, &env),
			URL:  wui.NewURLBuilder('h').SetZid(m.Zid).String(),
		})
	}
	return metas, nil
}
