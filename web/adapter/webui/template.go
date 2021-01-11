//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package webui provides wet-UI handlers for web requests.
package webui

import (
	"bytes"
	"context"
	"net/http"
	"sync"

	"zettelstore.de/z/auth/policy"
	"zettelstore.de/z/auth/token"
	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/place"
	"zettelstore.de/z/template"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/session"
)

type templatePlace interface {
	CanCreateZettel(ctx context.Context) bool
	GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error)
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
	SelectMeta(ctx context.Context, f *place.Filter, s *place.Sorter) ([]*meta.Meta, error)
	CanUpdateZettel(ctx context.Context, zettel domain.Zettel) bool
	AllowRenameZettel(ctx context.Context, zid id.Zid) bool
	CanDeleteZettel(ctx context.Context, zid id.Zid) bool
}

// TemplateEngine is the way to render HTML templates.
type TemplateEngine struct {
	place         templatePlace
	templateCache map[id.Zid]*template.Template
	mxCache       sync.RWMutex
	policy        policy.Policy

	stylesheetURL string
	homeURL       string
	listZettelURL string
	listRolesURL  string
	listTagsURL   string
	withAuth      bool
	loginURL      string
	reloadURL     string
	searchURL     string
}

// NewTemplateEngine creates a new TemplateEngine.
func NewTemplateEngine(p place.Place, pol policy.Policy) *TemplateEngine {
	te := &TemplateEngine{
		place:  p,
		policy: pol,

		stylesheetURL: adapter.NewURLBuilder('z').SetZid(
			id.BaseCSSZid).AppendQuery("_format", "raw").AppendQuery(
			"_part", "content").String(),
		homeURL:       adapter.NewURLBuilder('/').String(),
		listZettelURL: adapter.NewURLBuilder('h').String(),
		listRolesURL:  adapter.NewURLBuilder('k').SetZid(2).String(),
		listTagsURL:   adapter.NewURLBuilder('k').SetZid(3).String(),
		withAuth:      startup.WithAuth(),
		loginURL:      adapter.NewURLBuilder('a').String(),
		reloadURL:     adapter.NewURLBuilder('c').AppendQuery("_format", "html").String(),
		searchURL:     adapter.NewURLBuilder('s').String(),
	}
	te.observe(place.OnReload, id.Invalid)
	p.RegisterChangeObserver(te.observe)
	return te
}

func (te *TemplateEngine) observe(reason place.ChangeReason, zid id.Zid) {
	te.mxCache.Lock()
	if reason == place.OnReload || zid == id.BaseTemplateZid {
		te.templateCache = make(
			map[id.Zid]*template.Template, len(te.templateCache))
	} else {
		delete(te.templateCache, zid)
	}
	te.mxCache.Unlock()
}

func (te *TemplateEngine) cacheSetTemplate(zid id.Zid, t *template.Template) {
	te.mxCache.Lock()
	te.templateCache[zid] = t
	te.mxCache.Unlock()
}

func (te *TemplateEngine) cacheGetTemplate(zid id.Zid) (*template.Template, bool) {
	te.mxCache.RLock()
	t, ok := te.templateCache[zid]
	te.mxCache.RUnlock()
	return t, ok
}

func (te *TemplateEngine) canCreate(ctx context.Context, user *meta.Meta) bool {
	m := meta.New(id.Invalid)
	return te.policy.CanCreate(user, m) && te.place.CanCreateZettel(ctx)
}

func (te *TemplateEngine) canWrite(
	ctx context.Context, user *meta.Meta, zettel domain.Zettel) bool {
	return te.policy.CanWrite(user, zettel.Meta, zettel.Meta) &&
		te.place.CanUpdateZettel(ctx, zettel)
}

func (te *TemplateEngine) canRename(
	ctx context.Context, user *meta.Meta, m *meta.Meta) bool {
	return te.policy.CanRename(user, m) && te.place.AllowRenameZettel(ctx, m.Zid)
}

func (te *TemplateEngine) canDelete(
	ctx context.Context, user *meta.Meta, m *meta.Meta) bool {
	return te.policy.CanDelete(user, m) && te.place.CanDeleteZettel(ctx, m.Zid)
}

func (te *TemplateEngine) getTemplate(
	ctx context.Context, templateID id.Zid) (*template.Template, error) {
	if t, ok := te.cacheGetTemplate(templateID); ok {
		return t, nil
	}
	realTemplateZettel, err := te.place.GetZettel(ctx, templateID)
	if err != nil {
		return nil, err
	}
	t, err := template.ParseString(realTemplateZettel.Content.AsString(), nil)
	if err == nil {
		te.cacheSetTemplate(templateID, t)
	}
	return t, err
}

type simpleLink struct {
	Text string
	URL  string
}

type baseData struct {
	Lang           string
	MetaHeader     string
	StylesheetURL  string
	Title          string
	HomeURL        string
	ListZettelURL  string
	ListRolesURL   string
	ListTagsURL    string
	CanCreate      bool
	NewZettelURL   string
	NewZettelLinks []simpleLink
	WithAuth       bool
	UserIsValid    bool
	UserZettelURL  string
	UserIdent      string
	UserLogoutURL  string
	LoginURL       string
	CanReload      bool
	ReloadURL      string
	SearchURL      string
	Content        string
	FooterHTML     string
}

func (te *TemplateEngine) makeBaseData(
	ctx context.Context, lang string, title string, user *meta.Meta, data *baseData) {
	var (
		newZettelLinks []simpleLink
		userZettelURL  string
		userIdent      string
		userLogoutURL  string
	)
	canCreate := te.canCreate(ctx, user)
	if canCreate {
		newZettelLinks = te.fetchNewTemplates(ctx, user)
	}
	userIsValid := user != nil
	if userIsValid {
		userZettelURL = adapter.NewURLBuilder('h').SetZid(user.Zid).String()
		userIdent = user.GetDefault(meta.KeyUserID, "")
		userLogoutURL = adapter.NewURLBuilder('a').SetZid(user.Zid).String()
	}

	data.Lang = lang
	data.StylesheetURL = te.stylesheetURL
	data.Title = title
	data.HomeURL = te.homeURL
	data.ListZettelURL = te.listZettelURL
	data.ListRolesURL = te.listRolesURL
	data.ListTagsURL = te.listTagsURL
	data.CanCreate = canCreate
	data.NewZettelLinks = newZettelLinks
	data.WithAuth = te.withAuth
	data.UserIsValid = userIsValid
	data.UserZettelURL = userZettelURL
	data.UserIdent = userIdent
	data.UserLogoutURL = userLogoutURL
	data.LoginURL = te.loginURL
	data.CanReload = te.policy.CanReload(user)
	data.ReloadURL = te.reloadURL
	data.SearchURL = te.searchURL
	data.FooterHTML = runtime.GetFooterHTML()
}

// htmlAttrNewWindow eturns HTML attribute string for opening a link in a new window.
// If hasURL is false an empty string is returned.
func htmlAttrNewWindow(hasURL bool) string {
	if hasURL {
		return " target=\"_blank\" ref=\"noopener noreferrer\""
	}
	return ""
}

var templatePlaceFilter = &place.Filter{
	Expr: place.FilterExpr{
		meta.KeyRole: []string{meta.ValueRoleNewTemplate},
	},
}

var templatePlaceSorter = &place.Sorter{
	Order:      "id",
	Descending: false,
	Offset:     -1,
	Limit:      31, // Just to be one the safe side...
}

func (te *TemplateEngine) fetchNewTemplates(
	ctx context.Context, user *meta.Meta) []simpleLink {
	templateList, err := te.place.SelectMeta(ctx, templatePlaceFilter, templatePlaceSorter)
	if err != nil {
		return nil
	}
	result := make([]simpleLink, 0, len(templateList))
	for _, m := range templateList {
		if te.policy.CanRead(user, m) {
			title := runtime.GetTitle(m)
			langOption := encoder.StringOption{Key: "lang", Value: runtime.GetLang(m)}
			astTitle := parser.ParseInlines(
				input.NewInput(runtime.GetTitle(m)), meta.ValueSyntaxZmk)
			menuTitle, err := adapter.FormatInlines(astTitle, "html", &langOption)
			if err != nil {
				menuTitle, err = adapter.FormatInlines(astTitle, "text", &langOption)
				if err != nil {
					menuTitle = title
				}
			}
			result = append(result, simpleLink{
				Text: menuTitle,
				URL:  adapter.NewURLBuilder('n').SetZid(m.Zid).String(),
			})
		}
	}
	return result
}

func (te *TemplateEngine) renderTemplate(
	ctx context.Context,
	w http.ResponseWriter,
	templateID id.Zid,
	base *baseData,
	data interface{}) {

	bt, err := te.getTemplate(ctx, id.BaseTemplateZid)
	if err != nil {
		adapter.InternalServerError(w, "Unable to get base template", err)
		return
	}
	t, err := te.getTemplate(ctx, templateID)
	if err != nil {
		adapter.InternalServerError(w, "Unable to get template", err)
		return
	}
	if user := session.GetUser(ctx); user != nil {
		htmlLifetime, _ := startup.TokenLifetime()
		t, err := token.GetToken(user, htmlLifetime, token.KindHTML)
		if err == nil {
			session.SetToken(w, t, htmlLifetime)
		}
	}
	var content bytes.Buffer
	err = t.Render(&content, data)
	base.Content = content.String()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = bt.Render(w, base)
	if err != nil {
		adapter.InternalServerError(w, "Unable to render template", err)
	}
}
