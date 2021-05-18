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
	"zettelstore.de/z/parser"
	"zettelstore.de/z/place"
	"zettelstore.de/z/search"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/server"
)

// MakeListHTMLMetaHandler creates a HTTP handler for rendering the list of
// zettel as HTML.
func MakeListHTMLMetaHandler(
	ab server.AuthBuilder,
	te *TemplateEngine,
	listMeta usecase.ListMeta,
	listRole usecase.ListRole,
	listTags usecase.ListTags,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rwl := renderWebList{w: w, ab: ab, te: te}
		query := r.URL.Query()
		switch query.Get("_l") {
		case "r":
			rwl.renderRolesList(r, listRole)
		case "t":
			rwl.renderTagsList(r, listTags)
		default:
			rwl.renderZettelList(r, listMeta)
		}
	}
}

type renderWebList struct {
	w  http.ResponseWriter
	ab server.AuthBuilder
	te *TemplateEngine
}

func (rwl *renderWebList) renderZettelList(r *http.Request, listMeta usecase.ListMeta) {
	query := r.URL.Query()
	s := adapter.GetSearch(query, false)
	ctx := r.Context()
	title := listTitleSearch("Filter", s)
	rwl.renderMetaList(
		ctx, title, s,
		func(s *search.Search) ([]*meta.Meta, error) {
			if !s.HasComputedMetaKey() {
				ctx = place.NoEnrichContext(ctx)
			}
			return listMeta.Run(ctx, s)
		},
		func(offset int) string {
			return newPageURL(rwl.ab, 'h', query, offset, "_offset", "_limit")
		})
}

type roleInfo struct {
	Text string
	URL  string
}

func (rwl *renderWebList) renderRolesList(r *http.Request, listRole usecase.ListRole) {
	ctx := r.Context()
	roleList, err := listRole.Run(ctx)
	if err != nil {
		adapter.ReportUsecaseError(rwl.w, err)
		return
	}

	roleInfos := make([]roleInfo, 0, len(roleList))
	for _, role := range roleList {
		roleInfos = append(
			roleInfos,
			roleInfo{role, rwl.ab.NewURLBuilder('h').AppendQuery("role", role).String()})
	}

	user := rwl.ab.GetUser(ctx)
	var base baseData
	rwl.te.makeBaseData(ctx, runtime.GetDefaultLang(), runtime.GetSiteName(), user, &base)
	rwl.te.renderTemplate(ctx, rwl.w, id.RolesTemplateZid, &base, struct {
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

func (rwl *renderWebList) renderTagsList(r *http.Request, listTags usecase.ListTags) {
	ctx := r.Context()
	iMinCount, _ := strconv.Atoi(r.URL.Query().Get("min"))
	tagData, err := listTags.Run(ctx, iMinCount)
	if err != nil {
		rwl.te.reportError(ctx, rwl.w, err)
		return
	}

	user := rwl.ab.GetUser(ctx)
	tagsList := make([]tagInfo, 0, len(tagData))
	countMap := make(map[int]int)
	baseTagListURL := rwl.ab.NewURLBuilder('h')
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
	rwl.te.makeBaseData(ctx, runtime.GetDefaultLang(), runtime.GetSiteName(), user, &base)
	minCounts := make([]countInfo, 0, len(countList))
	for _, c := range countList {
		sCount := strconv.Itoa(c)
		minCounts = append(minCounts, countInfo{sCount, base.ListTagsURL + "&min=" + sCount})
	}

	rwl.te.renderTemplate(ctx, rwl.w, id.TagsTemplateZid, &base, struct {
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
	ab server.AuthBuilder,
	te *TemplateEngine,
	ucSearch usecase.Search,
	getMeta usecase.GetMeta,
	getZettel usecase.GetZettel,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		ctx := r.Context()
		s := adapter.GetSearch(query, true)
		if s == nil {
			redirectFound(w, r, ab.NewURLBuilder('h'))
			return
		}

		rwl := renderWebList{w: w, ab: ab, te: te}
		title := listTitleSearch("Search", s)
		rwl.renderMetaList(
			ctx, title, s, func(s *search.Search) ([]*meta.Meta, error) {
				if !s.HasComputedMetaKey() {
					ctx = place.NoEnrichContext(ctx)
				}
				return ucSearch.Run(ctx, s)
			},
			func(offset int) string {
				return newPageURL(ab, 'f', query, offset, "offset", "limit")
			})
	}
}

// MakeZettelContextHandler creates a new HTTP handler for the use case "zettel context".
func MakeZettelContextHandler(
	ab server.AuthBuilder, te *TemplateEngine, getContext usecase.ZettelContext) http.HandlerFunc {
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
		metaLinks, err := buildHTMLMetaList(ab, metaList)
		if err != nil {
			adapter.InternalServerError(w, "Build HTML meta list", err)
			return
		}

		depths := []string{"2", "3", "4", "5", "6", "7", "8", "9", "10"}
		depthLinks := make([]simpleLink, len(depths))
		depthURL := ab.NewURLBuilder('j').SetZid(zid)
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
		user := ab.GetUser(ctx)
		te.makeBaseData(ctx, runtime.GetDefaultLang(), runtime.GetSiteName(), user, &base)
		te.renderTemplate(ctx, w, id.ContextTemplateZid, &base, struct {
			Title   string
			InfoURL string
			Depths  []simpleLink
			Start   simpleLink
			Metas   []simpleLink
		}{
			Title:   "Zettel Context",
			InfoURL: ab.NewURLBuilder('i').SetZid(zid).String(),
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

func (rwl *renderWebList) renderMetaList(
	ctx context.Context,
	title string,
	s *search.Search,
	ucMetaList func(sorter *search.Search) ([]*meta.Meta, error),
	pageURL func(int) string) {

	var metaList []*meta.Meta
	var err error
	var prevURL, nextURL string
	if lps := runtime.GetListPageSize(); lps > 0 {
		if s.GetLimit() < lps {
			s.SetLimit(lps + 1)
		}

		metaList, err = ucMetaList(s)
		if err != nil {
			rwl.te.reportError(ctx, rwl.w, err)
			return
		}
		if offset := s.GetOffset(); offset > 0 {
			offset -= lps
			if offset < 0 {
				offset = 0
			}
			prevURL = pageURL(offset)
		}
		if len(metaList) >= s.GetLimit() {
			nextURL = pageURL(s.GetOffset() + lps)
			metaList = metaList[:len(metaList)-1]
		}
	} else {
		metaList, err = ucMetaList(s)
		if err != nil {
			rwl.te.reportError(ctx, rwl.w, err)
			return
		}
	}
	user := rwl.ab.GetUser(ctx)
	metas, err := buildHTMLMetaList(rwl.ab, metaList)
	if err != nil {
		rwl.te.reportError(ctx, rwl.w, err)
		return
	}
	var base baseData
	rwl.te.makeBaseData(ctx, runtime.GetDefaultLang(), runtime.GetSiteName(), user, &base)
	rwl.te.renderTemplate(ctx, rwl.w, id.ListTemplateZid, &base, struct {
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

func listTitleSearch(prefix string, s *search.Search) string {
	if s == nil {
		return runtime.GetSiteName()
	}
	var sb strings.Builder
	sb.WriteString(prefix)
	if s != nil {
		sb.WriteString(": ")
		s.Print(&sb)
	}
	return sb.String()
}

func newPageURL(b server.Builder, key byte, query url.Values, offset int, offsetKey, limitKey string) string {
	ub := b.NewURLBuilder(key)
	for key, values := range query {
		if key != offsetKey && key != limitKey {
			for _, val := range values {
				ub.AppendQuery(key, val)
			}
		}
	}
	if offset > 0 {
		ub.AppendQuery(offsetKey, strconv.Itoa(offset))
	}
	return ub.String()
}

// buildHTMLMetaList builds a zettel list based on a meta list for HTML rendering.
func buildHTMLMetaList(b server.Builder, metaList []*meta.Meta) ([]simpleLink, error) {
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
		env := encoder.Environment{Lang: lang, Interactive: true}
		htmlTitle, err := adapter.FormatInlines(parser.ParseMetadata(title), "html", &env)
		if err != nil {
			return nil, err
		}
		metas = append(metas, simpleLink{
			Text: htmlTitle,
			URL:  b.NewURLBuilder('h').SetZid(m.Zid).String(),
		})
	}
	return metas, nil
}
