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
	"bytes"
	"context"
	"log"
	"net/http"
	"sync"

	"zettelstore.de/z/auth/policy"
	"zettelstore.de/z/auth/token"
	"zettelstore.de/z/collect"
	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/index"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/change"
	"zettelstore.de/z/search"
	"zettelstore.de/z/template"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/session"
)

type templatePlace interface {
	CanCreateZettel(ctx context.Context) bool
	GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error)
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
	SelectMeta(ctx context.Context, f *search.Filter, s *search.Sorter) ([]*meta.Meta, error)
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
	searchURL     string
}

// NewTemplateEngine creates a new TemplateEngine.
func NewTemplateEngine(mgr place.Manager, pol policy.Policy) *TemplateEngine {
	te := &TemplateEngine{
		place:  mgr,
		policy: pol,

		stylesheetURL: adapter.NewURLBuilder('z').SetZid(
			id.BaseCSSZid).AppendQuery("_format", "raw").AppendQuery(
			"_part", "content").String(),
		homeURL:       adapter.NewURLBuilder('/').String(),
		listZettelURL: adapter.NewURLBuilder('h').String(),
		listRolesURL:  adapter.NewURLBuilder('h').AppendQuery("_l", "r").String(),
		listTagsURL:   adapter.NewURLBuilder('h').AppendQuery("_l", "t").String(),
		withAuth:      startup.WithAuth(),
		loginURL:      adapter.NewURLBuilder('a').String(),
		searchURL:     adapter.NewURLBuilder('f').String(),
	}
	te.observe(change.Info{Reason: change.OnReload, Zid: id.Invalid})
	mgr.RegisterObserver(te.observe)
	return te
}

func (te *TemplateEngine) observe(ci change.Info) {
	te.mxCache.Lock()
	if ci.Reason == change.OnReload || ci.Zid == id.BaseTemplateZid {
		te.templateCache = make(map[id.Zid]*template.Template, len(te.templateCache))
	} else {
		delete(te.templateCache, ci.Zid)
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
	ctx context.Context, user, meta *meta.Meta, content domain.Content) bool {
	return te.policy.CanWrite(user, meta, meta) &&
		te.place.CanUpdateZettel(ctx, domain.Zettel{Meta: meta, Content: content})
}

func (te *TemplateEngine) canRename(ctx context.Context, user, m *meta.Meta) bool {
	return te.policy.CanRename(user, m) && te.place.AllowRenameZettel(ctx, m.Zid)
}

func (te *TemplateEngine) canDelete(ctx context.Context, user, m *meta.Meta) bool {
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
		// t.SetErrorOnMissing()
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
	WithUser       bool
	WithAuth       bool
	UserIsValid    bool
	UserZettelURL  string
	UserIdent      string
	UserLogoutURL  string
	LoginURL       string
	ListZettelURL  string
	ListRolesURL   string
	ListTagsURL    string
	CanCreate      bool
	NewZettelURL   string
	NewZettelLinks []simpleLink
	SearchURL      string
	Content        string
	FooterHTML     string
}

func (te *TemplateEngine) makeBaseData(
	ctx context.Context, lang, title string, user *meta.Meta, data *baseData) {
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
	data.WithAuth = te.withAuth
	data.WithUser = data.WithAuth
	data.UserIsValid = userIsValid
	data.UserZettelURL = userZettelURL
	data.UserIdent = userIdent
	data.UserLogoutURL = userLogoutURL
	data.LoginURL = te.loginURL
	data.ListZettelURL = te.listZettelURL
	data.ListRolesURL = te.listRolesURL
	data.ListTagsURL = te.listTagsURL
	data.CanCreate = canCreate
	data.NewZettelLinks = newZettelLinks
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

func (te *TemplateEngine) fetchNewTemplates(ctx context.Context, user *meta.Meta) []simpleLink {
	ctx = index.NoEnrichContext(ctx)
	menu, err := te.place.GetZettel(ctx, id.TOCNewTemplateZid)
	if err != nil {
		return nil
	}
	zn := parser.ParseZettel(menu, "")
	refs := collect.Order(zn)
	result := make([]simpleLink, 0, len(refs))
	for _, ref := range refs {
		zid, err := id.Parse(ref.URL.Path)
		if err != nil {
			continue
		}
		m, err := te.place.GetMeta(ctx, zid)
		if err != nil {
			continue
		}
		if !te.policy.CanRead(user, m) {
			continue
		}
		title := runtime.GetTitle(m)
		astTitle := parser.ParseInlines(input.NewInput(runtime.GetTitle(m)), meta.ValueSyntaxZmk)
		env := encoder.Environment{Lang: runtime.GetLang(m)}
		menuTitle, err := adapter.FormatInlines(astTitle, "html", &env)
		if err != nil {
			menuTitle, err = adapter.FormatInlines(astTitle, "text", nil)
			if err != nil {
				menuTitle = title
			}
		}
		result = append(result, simpleLink{
			Text: menuTitle,
			URL:  adapter.NewURLBuilder('g').SetZid(m.Zid).String(),
		})
	}
	return result
}

func (te *TemplateEngine) renderTemplate(
	ctx context.Context,
	w http.ResponseWriter,
	templateID id.Zid,
	base *baseData,
	data interface{}) {
	te.renderTemplateStatus(ctx, w, http.StatusOK, templateID, base, data)
}

func (te *TemplateEngine) reportError(ctx context.Context, w http.ResponseWriter, err error) {
	code, text := adapter.CodeMessageFromError(err)
	if code == http.StatusInternalServerError {
		log.Printf("%v: %v", text, err)
	}
	user := session.GetUser(ctx)
	var base baseData
	te.makeBaseData(ctx, "en", "Error", user, &base)
	te.renderTemplateStatus(ctx, w, code, id.ErrorTemplateZid, &base, struct {
		ErrorTitle string
		ErrorText  string
	}{
		ErrorTitle: http.StatusText(code),
		ErrorText:  text,
	})
}

func (te *TemplateEngine) renderTemplateStatus(
	ctx context.Context,
	w http.ResponseWriter,
	code int,
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
		if tok, err1 := token.GetToken(user, htmlLifetime, token.KindHTML); err1 == nil {
			session.SetToken(w, tok, htmlLifetime)
		}
	}
	var content bytes.Buffer
	err = t.Render(&content, data)
	if err == nil {
		base.Content = content.String()
		w.Header().Set(adapter.ContentType, "text/html; charset=utf-8")
		w.WriteHeader(code)
		err = bt.Render(w, base)
	}
	if err != nil {
		log.Println("Unable to render template", err)
	}
}
