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

	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/index"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/place"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/session"
)

// MakeListHTMLMetaHandler creates a HTTP handler for rendering the list of
// zettel as HTML.
func MakeListHTMLMetaHandler(
	te *TemplateEngine,
	listMeta usecase.ListMeta,
	listRole usecase.ListRole,
	listTags usecase.ListTags,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		switch query.Get("_l") {
		case "r":
			renderWebUIRolesList(w, r, te, listRole)
		case "t":
			renderWebUITagsList(w, r, te, listTags)
		default:
			renderWebUIZettelList(w, r, te, listMeta)
		}
	}
}

func renderWebUIZettelList(
	w http.ResponseWriter, r *http.Request, te *TemplateEngine, listMeta usecase.ListMeta) {
	query := r.URL.Query()
	filter, sorter := adapter.GetFilterSorter(query, false)
	ctx := r.Context()
	title := listTitleFilterSorter("Filter", filter, sorter)
	renderWebUIMetaList(
		ctx, w, te, title, sorter,
		func(sorter *place.Sorter) ([]*meta.Meta, error) {
			if filter == nil && (sorter == nil || sorter.Order == "") {
				ctx = index.NoEnrichContext(ctx)
			}
			return listMeta.Run(ctx, filter, sorter)
		},
		func(offset int) string {
			return newPageURL('h', query, offset, "_offset", "_limit")
		})
}

type roleInfo struct {
	Text string
	URL  string
}

func renderWebUIRolesList(
	w http.ResponseWriter,
	r *http.Request,
	te *TemplateEngine,
	listRole usecase.ListRole,
) {
	ctx := r.Context()
	roleList, err := listRole.Run(ctx)
	if err != nil {
		adapter.ReportUsecaseError(w, err)
		return
	}

	roleInfos := make([]roleInfo, 0, len(roleList))
	for _, r := range roleList {
		roleInfos = append(
			roleInfos,
			roleInfo{r, adapter.NewURLBuilder('h').AppendQuery("role", r).String()})
	}

	user := session.GetUser(ctx)
	var base baseData
	te.makeBaseData(ctx, runtime.GetDefaultLang(), runtime.GetSiteName(), user, &base)
	te.renderTemplate(ctx, w, id.RolesTemplateZid, &base, struct {
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
	Name  string
	URL   string
	count int
	Count string
	Size  string
}

var fontSizes = [...]int{75, 83, 100, 117, 150, 200}

func renderWebUITagsList(
	w http.ResponseWriter,
	r *http.Request,
	te *TemplateEngine,
	listTags usecase.ListTags,
) {
	ctx := r.Context()
	iMinCount, _ := strconv.Atoi(r.URL.Query().Get("min"))
	tagData, err := listTags.Run(ctx, iMinCount)
	if err != nil {
		te.reportError(ctx, w, err)
		return
	}

	user := session.GetUser(ctx)
	tagsList := make([]tagInfo, 0, len(tagData))
	countMap := make(map[int]int)
	baseTagListURL := adapter.NewURLBuilder('h')
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
		count := tagsList[i].count
		tagsList[i].Count = strconv.Itoa(count)
		tagsList[i].Size = strconv.Itoa(countMap[count])
	}

	var base baseData
	te.makeBaseData(ctx, runtime.GetDefaultLang(), runtime.GetSiteName(), user, &base)
	minCounts := make([]countInfo, 0, len(countList))
	for _, c := range countList {
		sCount := strconv.Itoa(c)
		minCounts = append(minCounts, countInfo{sCount, base.ListTagsURL + "&min=" + sCount})
	}

	te.renderTemplate(ctx, w, id.TagsTemplateZid, &base, struct {
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
func MakeSearchHandler(
	te *TemplateEngine,
	search usecase.Search,
	getMeta usecase.GetMeta,
	getZettel usecase.GetZettel,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		filter, sorter := adapter.GetFilterSorter(query, true)
		if filter == nil {
			redirectFound(w, r, adapter.NewURLBuilder('h'))
			return
		}

		ctx := r.Context()
		title := listTitleFilterSorter("Search", filter, sorter)
		renderWebUIMetaList(
			ctx, w, te, title, sorter,
			func(sorter *place.Sorter) ([]*meta.Meta, error) {
				if filter == nil && (sorter == nil || sorter.Order == "") {
					ctx = index.NoEnrichContext(ctx)
				}
				return search.Run(ctx, filter, sorter)
			},
			func(offset int) string {
				return newPageURL('f', query, offset, "offset", "limit")
			})
	}
}

// MakeZettelContextHandler creates a new HTTP handler for the use case "zettel context".
func MakeZettelContextHandler(te *TemplateEngine, getContext usecase.ZettelContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			te.reportError(ctx, w, place.ErrNotFound)
			return
		}
		q := r.URL.Query()
		dir := usecase.ParseZCDirection(q.Get("dir"))
		depth := getIntParameter(q, "depth", 5)
		limit := getIntParameter(q, "limit", 200)
		metaList, err := getContext.Run(ctx, zid, dir, depth, limit)
		if err != nil {
			te.reportError(ctx, w, err)
			return
		}
		metaLinks, err := buildHTMLMetaList(metaList)
		if err != nil {
			adapter.InternalServerError(w, "Build HTML meta list", err)
			return
		}

		depths := []string{"2", "3", "4", "5", "6", "7", "8", "9", "10"}
		depthLinks := make([]simpleLink, len(depths))
		depthURL := adapter.NewURLBuilder('j').SetZid(zid)
		for i, depth := range depths {
			depthURL.ClearQuery()
			switch dir {
			case usecase.ZettelContextBackward:
				depthURL.AppendQuery("dir", "backward")
			case usecase.ZettelContextForward:
				depthURL.AppendQuery("dir", "forward")
			}
			depthURL.AppendQuery("depth", depth)
			depthLinks[i].Text = depth
			depthLinks[i].URL = depthURL.String()
		}
		var base baseData
		user := session.GetUser(ctx)
		te.makeBaseData(ctx, runtime.GetDefaultLang(), runtime.GetSiteName(), user, &base)
		te.renderTemplate(ctx, w, id.ContextTemplateZid, &base, struct {
			Title   string
			InfoURL string
			Depths  []simpleLink
			Start   simpleLink
			Metas   []simpleLink
		}{
			Title:   "Zettel Context",
			InfoURL: adapter.NewURLBuilder('i').SetZid(zid).String(),
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

func renderWebUIMetaList(
	ctx context.Context, w http.ResponseWriter, te *TemplateEngine,
	title string,
	sorter *place.Sorter,
	ucMetaList func(sorter *place.Sorter) ([]*meta.Meta, error),
	pageURL func(int) string) {

	var metaList []*meta.Meta
	var err error
	var prevURL, nextURL string
	if lps := runtime.GetListPageSize(); lps > 0 {
		sorter = sorter.Ensure()
		if sorter.Limit < lps {
			sorter.Limit = lps + 1
		}

		metaList, err = ucMetaList(sorter)
		if err != nil {
			te.reportError(ctx, w, err)
			return
		}
		if offset := sorter.Offset; offset > 0 {
			offset -= lps
			if offset < 0 {
				offset = 0
			}
			prevURL = pageURL(offset)
		}
		if len(metaList) >= sorter.Limit {
			nextURL = pageURL(sorter.Offset + lps)
			metaList = metaList[:len(metaList)-1]
		}
	} else {
		metaList, err = ucMetaList(sorter)
		if err != nil {
			te.reportError(ctx, w, err)
			return
		}
	}
	user := session.GetUser(ctx)
	metas, err := buildHTMLMetaList(metaList)
	if err != nil {
		te.reportError(ctx, w, err)
		return
	}
	var base baseData
	te.makeBaseData(ctx, runtime.GetDefaultLang(), runtime.GetSiteName(), user, &base)
	te.renderTemplate(ctx, w, id.ListTemplateZid, &base, struct {
		Title       string
		Metas       []simpleLink
		HasPrevNext bool
		HasPrev     bool
		PrevURL     string
		HasNext     bool
		NextURL     string
	}{
		Title:       title,
		Metas:       metas,
		HasPrevNext: len(prevURL) > 0 || len(nextURL) > 0,
		HasPrev:     len(prevURL) > 0,
		PrevURL:     prevURL,
		HasNext:     len(nextURL) > 0,
		NextURL:     nextURL,
	})
}

func listTitleFilterSorter(prefix string, filter *place.Filter, sorter *place.Sorter) string {
	if filter == nil && sorter == nil {
		return runtime.GetSiteName()
	}
	var sb strings.Builder
	sb.WriteString(prefix)
	sb.WriteString(": ")
	if filter != nil {
		filter.Print(&sb)
		if sorter != nil {
			sb.WriteString(" | ")
			sorter.Print(&sb)
		}
	} else if sorter != nil {
		sorter.Print(&sb)
	}
	return sb.String()
}

func newPageURL(key byte, query url.Values, offset int, offsetKey, limitKey string) string {
	urlBuilder := adapter.NewURLBuilder(key)
	for key, values := range query {
		if key != offsetKey && key != limitKey {
			for _, val := range values {
				urlBuilder.AppendQuery(key, val)
			}
		}
	}
	if offset > 0 {
		urlBuilder.AppendQuery(offsetKey, strconv.Itoa(offset))
	}
	return urlBuilder.String()
}

// buildHTMLMetaList builds a zettel list based on a meta list for HTML rendering.
func buildHTMLMetaList(metaList []*meta.Meta) ([]simpleLink, error) {
	defaultLang := runtime.GetDefaultLang()
	metas := make([]simpleLink, 0, len(metaList))
	for _, m := range metaList {
		var lang string
		if val, ok := m.Get(meta.KeyLang); ok {
			lang = val
		} else {
			lang = defaultLang
		}
		title, _ := m.Get(meta.KeyTitle)
		env := encoder.Environment{Lang: lang}
		htmlTitle, err := adapter.FormatInlines(parser.ParseTitle(title), "html", &env)
		if err != nil {
			return nil, err
		}
		metas = append(metas, simpleLink{
			Text: htmlTitle,
			URL:  adapter.NewURLBuilder('h').SetZid(m.Zid).String(),
		})
	}
	return metas, nil
}
