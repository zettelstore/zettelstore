//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
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
	"time"

	"zettelstore.de/z/auth"
	"zettelstore.de/z/collect"
	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/change"
	"zettelstore.de/z/service"
	"zettelstore.de/z/template"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/server"
)

// WebUI holds all data for delivering the web ui.
type WebUI struct {
	ab     server.AuthBuilder
	authz  auth.AuthzManager
	token  auth.TokenManager
	place  webuiPlace
	policy auth.Policy

	templateCache map[id.Zid]*template.Template
	mxCache       sync.RWMutex

	tokenLifetime time.Duration
	stylesheetURL string
	homeURL       string
	listZettelURL string
	listRolesURL  string
	listTagsURL   string
	withAuth      bool
	loginURL      string
	searchURL     string
}

type webuiPlace interface {
	CanCreateZettel(ctx context.Context) bool
	GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error)
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
	CanUpdateZettel(ctx context.Context, zettel domain.Zettel) bool
	AllowRenameZettel(ctx context.Context, zid id.Zid) bool
	CanDeleteZettel(ctx context.Context, zid id.Zid) bool
}

// New creates a new WebUI struct.
func New(ab server.AuthBuilder, authz auth.AuthzManager, token auth.TokenManager,
	mgr place.Manager, pol auth.Policy) *WebUI {
	wui := &WebUI{
		ab:     ab,
		authz:  authz,
		token:  token,
		place:  mgr,
		policy: pol,

		tokenLifetime: service.Main.GetConfig(service.SubWeb, service.WebTokenLifetimeHTML).(time.Duration),
		stylesheetURL: ab.NewURLBuilder('z').SetZid(
			id.BaseCSSZid).AppendQuery("_format", "raw").AppendQuery(
			"_part", "content").String(),
		homeURL:       ab.NewURLBuilder('/').String(),
		listZettelURL: ab.NewURLBuilder('h').String(),
		listRolesURL:  ab.NewURLBuilder('h').AppendQuery("_l", "r").String(),
		listTagsURL:   ab.NewURLBuilder('h').AppendQuery("_l", "t").String(),
		withAuth:      authz.WithAuth(),
		loginURL:      ab.NewURLBuilder('a').String(),
		searchURL:     ab.NewURLBuilder('f').String(),
	}
	wui.observe(change.Info{Reason: change.OnReload, Zid: id.Invalid})
	mgr.RegisterObserver(wui.observe)
	return wui
}

func (wui *WebUI) observe(ci change.Info) {
	wui.mxCache.Lock()
	if ci.Reason == change.OnReload || ci.Zid == id.BaseTemplateZid {
		wui.templateCache = make(map[id.Zid]*template.Template, len(wui.templateCache))
	} else {
		delete(wui.templateCache, ci.Zid)
	}
	wui.mxCache.Unlock()
}

func (wui *WebUI) cacheSetTemplate(zid id.Zid, t *template.Template) {
	wui.mxCache.Lock()
	wui.templateCache[zid] = t
	wui.mxCache.Unlock()
}

func (wui *WebUI) cacheGetTemplate(zid id.Zid) (*template.Template, bool) {
	wui.mxCache.RLock()
	t, ok := wui.templateCache[zid]
	wui.mxCache.RUnlock()
	return t, ok
}

func (wui *WebUI) canCreate(ctx context.Context, user *meta.Meta) bool {
	m := meta.New(id.Invalid)
	return wui.policy.CanCreate(user, m) && wui.place.CanCreateZettel(ctx)
}

func (wui *WebUI) canWrite(
	ctx context.Context, user, meta *meta.Meta, content domain.Content) bool {
	return wui.policy.CanWrite(user, meta, meta) &&
		wui.place.CanUpdateZettel(ctx, domain.Zettel{Meta: meta, Content: content})
}

func (wui *WebUI) canRename(ctx context.Context, user, m *meta.Meta) bool {
	return wui.policy.CanRename(user, m) && wui.place.AllowRenameZettel(ctx, m.Zid)
}

func (wui *WebUI) canDelete(ctx context.Context, user, m *meta.Meta) bool {
	return wui.policy.CanDelete(user, m) && wui.place.CanDeleteZettel(ctx, m.Zid)
}

func (wui *WebUI) getTemplate(
	ctx context.Context, templateID id.Zid) (*template.Template, error) {
	if t, ok := wui.cacheGetTemplate(templateID); ok {
		return t, nil
	}
	realTemplateZettel, err := wui.place.GetZettel(ctx, templateID)
	if err != nil {
		return nil, err
	}
	t, err := template.ParseString(realTemplateZettel.Content.AsString(), nil)
	if err == nil {
		// t.SetErrorOnMissing()
		wui.cacheSetTemplate(templateID, t)
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

func (wui *WebUI) makeBaseData(
	ctx context.Context, lang, title string, user *meta.Meta, data *baseData) {
	var (
		newZettelLinks []simpleLink
		userZettelURL  string
		userIdent      string
		userLogoutURL  string
	)
	canCreate := wui.canCreate(ctx, user)
	if canCreate {
		newZettelLinks = wui.fetchNewTemplates(ctx, user)
	}
	userIsValid := user != nil
	if userIsValid {
		userZettelURL = wui.ab.NewURLBuilder('h').SetZid(user.Zid).String()
		userIdent = user.GetDefault(meta.KeyUserID, "")
		userLogoutURL = wui.ab.NewURLBuilder('a').SetZid(user.Zid).String()
	}

	data.Lang = lang
	data.StylesheetURL = wui.stylesheetURL
	data.Title = title
	data.HomeURL = wui.homeURL
	data.WithAuth = wui.withAuth
	data.WithUser = data.WithAuth
	data.UserIsValid = userIsValid
	data.UserZettelURL = userZettelURL
	data.UserIdent = userIdent
	data.UserLogoutURL = userLogoutURL
	data.LoginURL = wui.loginURL
	data.ListZettelURL = wui.listZettelURL
	data.ListRolesURL = wui.listRolesURL
	data.ListTagsURL = wui.listTagsURL
	data.CanCreate = canCreate
	data.NewZettelLinks = newZettelLinks
	data.SearchURL = wui.searchURL
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

func (wui *WebUI) fetchNewTemplates(ctx context.Context, user *meta.Meta) []simpleLink {
	ctx = place.NoEnrichContext(ctx)
	menu, err := wui.place.GetZettel(ctx, id.TOCNewTemplateZid)
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
		m, err := wui.place.GetMeta(ctx, zid)
		if err != nil {
			continue
		}
		if !wui.policy.CanRead(user, m) {
			continue
		}
		title := runtime.GetTitle(m)
		astTitle := parser.ParseInlines(input.NewInput(title), meta.ValueSyntaxZmk)
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
			URL:  wui.ab.NewURLBuilder('g').SetZid(m.Zid).String(),
		})
	}
	return result
}

func (wui *WebUI) renderTemplate(
	ctx context.Context,
	w http.ResponseWriter,
	templateID id.Zid,
	base *baseData,
	data interface{}) {
	wui.renderTemplateStatus(ctx, w, http.StatusOK, templateID, base, data)
}

func (wui *WebUI) reportError(ctx context.Context, w http.ResponseWriter, err error) {
	code, text := adapter.CodeMessageFromError(err)
	if code == http.StatusInternalServerError {
		log.Printf("%v: %v", text, err)
	}
	user := wui.ab.GetUser(ctx)
	var base baseData
	wui.makeBaseData(ctx, meta.ValueLangEN, "Error", user, &base)
	wui.renderTemplateStatus(ctx, w, code, id.ErrorTemplateZid, &base, struct {
		ErrorTitle string
		ErrorText  string
	}{
		ErrorTitle: http.StatusText(code),
		ErrorText:  text,
	})
}

func (wui *WebUI) renderTemplateStatus(
	ctx context.Context,
	w http.ResponseWriter,
	code int,
	templateID id.Zid,
	base *baseData,
	data interface{}) {

	bt, err := wui.getTemplate(ctx, id.BaseTemplateZid)
	if err != nil {
		adapter.InternalServerError(w, "Unable to get base template", err)
		return
	}
	t, err := wui.getTemplate(ctx, templateID)
	if err != nil {
		adapter.InternalServerError(w, "Unable to get template", err)
		return
	}
	if user := wui.ab.GetUser(ctx); user != nil {
		if tok, err1 := wui.token.GetToken(user, wui.tokenLifetime, auth.KindHTML); err1 == nil {
			wui.ab.SetToken(w, tok, wui.tokenLifetime)
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

func (wui *WebUI) getUser(ctx context.Context) *meta.Meta   { return wui.ab.GetUser(ctx) }
func (wui *WebUI) newURLBuilder(key byte) server.URLBuilder { return wui.ab.NewURLBuilder(key) }
func (wui *WebUI) clearToken(ctx context.Context, w http.ResponseWriter) context.Context {
	return wui.ab.ClearToken(ctx, w)
}
func (wui *WebUI) setToken(w http.ResponseWriter, token []byte) {
	wui.ab.SetToken(w, token, wui.tokenLifetime)
}
