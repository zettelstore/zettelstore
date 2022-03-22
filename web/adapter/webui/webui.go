//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
//
// This file is part of Zettelstore.
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
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"zettelstore.de/c/api"
	"zettelstore.de/z/auth"
	"zettelstore.de/z/box"
	"zettelstore.de/z/collect"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/logger"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/template"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/server"
)

// WebUI holds all data for delivering the web ui.
type WebUI struct {
	log      *logger.Logger
	debug    bool
	ab       server.AuthBuilder
	authz    auth.AuthzManager
	rtConfig config.Config
	token    auth.TokenManager
	box      webuiBox
	policy   auth.Policy

	mxCache       sync.RWMutex
	templateCache map[id.Zid]*template.Template

	mxRoleCSSMap sync.RWMutex
	roleCSSMap   map[string]id.Zid

	tokenLifetime time.Duration
	cssBaseURL    string
	cssUserURL    string
	homeURL       string
	listZettelURL string
	listRolesURL  string
	listTagsURL   string
	refreshURL    string
	withAuth      bool
	loginURL      string
	logoutURL     string
	searchURL     string
}

type webuiBox interface {
	CanCreateZettel(ctx context.Context) bool
	GetZettel(ctx context.Context, zid id.Zid) (domain.Zettel, error)
	GetMeta(ctx context.Context, zid id.Zid) (*meta.Meta, error)
	CanUpdateZettel(ctx context.Context, zettel domain.Zettel) bool
	AllowRenameZettel(ctx context.Context, zid id.Zid) bool
	CanDeleteZettel(ctx context.Context, zid id.Zid) bool
}

// New creates a new WebUI struct.
func New(log *logger.Logger, ab server.AuthBuilder, authz auth.AuthzManager, rtConfig config.Config, token auth.TokenManager,
	mgr box.Manager, pol auth.Policy) *WebUI {
	loginoutBase := ab.NewURLBuilder('i')
	wui := &WebUI{
		log:      log,
		debug:    kernel.Main.GetConfig(kernel.CoreService, kernel.CoreDebug).(bool),
		ab:       ab,
		rtConfig: rtConfig,
		authz:    authz,
		token:    token,
		box:      mgr,
		policy:   pol,

		tokenLifetime: kernel.Main.GetConfig(kernel.WebService, kernel.WebTokenLifetimeHTML).(time.Duration),
		cssBaseURL:    ab.NewURLBuilder('z').SetZid(api.ZidBaseCSS).String(),
		cssUserURL:    ab.NewURLBuilder('z').SetZid(api.ZidUserCSS).String(),
		homeURL:       ab.NewURLBuilder('/').String(),
		listZettelURL: ab.NewURLBuilder('h').String(),
		listRolesURL:  ab.NewURLBuilder('h').AppendQuery("_l", "r").String(),
		listTagsURL:   ab.NewURLBuilder('h').AppendQuery("_l", "t").String(),
		refreshURL:    ab.NewURLBuilder('g').AppendQuery("_c", "r").String(),
		withAuth:      authz.WithAuth(),
		loginURL:      loginoutBase.String(),
		logoutURL:     loginoutBase.AppendQuery("logout", "").String(),
		searchURL:     ab.NewURLBuilder('h').String(),
	}
	wui.observe(box.UpdateInfo{Box: mgr, Reason: box.OnReload, Zid: id.Invalid})
	mgr.RegisterObserver(wui.observe)
	return wui
}

func (wui *WebUI) observe(ci box.UpdateInfo) {
	wui.mxCache.Lock()
	if ci.Reason == box.OnReload || ci.Zid == id.BaseTemplateZid {
		wui.templateCache = make(map[id.Zid]*template.Template, len(wui.templateCache))
	} else {
		delete(wui.templateCache, ci.Zid)
	}
	wui.mxCache.Unlock()
	wui.mxRoleCSSMap.Lock()
	if ci.Reason == box.OnReload || ci.Zid == id.RoleCSSMapZid {
		wui.roleCSSMap = nil
	}
	wui.mxRoleCSSMap.Unlock()
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

func (wui *WebUI) retrieveCSSZidFromRole(ctx context.Context, m meta.Meta) (id.Zid, error) {
	wui.mxRoleCSSMap.RLock()
	if wui.roleCSSMap == nil {
		wui.mxRoleCSSMap.RUnlock()
		wui.mxRoleCSSMap.Lock()
		mMap, err := wui.box.GetMeta(ctx, id.RoleCSSMapZid)
		if err == nil {
			wui.roleCSSMap = make(map[string]id.Zid, len(wui.roleCSSMap))
			for _, p := range mMap.PairsRest() {
				key := p.Key
				if len(key) < 5 || !strings.HasPrefix(key, "css-") {
					continue
				}
				zid, err2 := id.Parse(p.Value)
				if err2 != nil {
					continue
				}
				wui.roleCSSMap[p.Key[4:]] = zid
			}
		}
		wui.mxRoleCSSMap.Unlock()
		if err != nil {
			return id.Invalid, err
		}
		wui.mxRoleCSSMap.RLock()
	}

	defer wui.mxRoleCSSMap.RUnlock()
	if role, found := m.Get("css-role"); found {
		if result, found2 := wui.roleCSSMap[role]; found2 {
			return result, nil
		}
	}
	if role, found := m.Get(api.KeyRole); found {
		if result, found2 := wui.roleCSSMap[role]; found2 {
			return result, nil
		}
	}
	return id.Invalid, nil
}

func (wui *WebUI) canCreate(ctx context.Context, user *meta.Meta) bool {
	m := meta.New(id.Invalid)
	return wui.policy.CanCreate(user, m) && wui.box.CanCreateZettel(ctx)
}

func (wui *WebUI) canWrite(
	ctx context.Context, user, meta *meta.Meta, content domain.Content) bool {
	return wui.policy.CanWrite(user, meta, meta) &&
		wui.box.CanUpdateZettel(ctx, domain.Zettel{Meta: meta, Content: content})
}

func (wui *WebUI) canRename(ctx context.Context, user, m *meta.Meta) bool {
	return wui.policy.CanRename(user, m) && wui.box.AllowRenameZettel(ctx, m.Zid)
}

func (wui *WebUI) canDelete(ctx context.Context, user, m *meta.Meta) bool {
	return wui.policy.CanDelete(user, m) && wui.box.CanDeleteZettel(ctx, m.Zid)
}

func (wui *WebUI) canRefresh(user *meta.Meta) bool {
	return wui.policy.CanRefresh(user)
}

func (wui *WebUI) getTemplate(
	ctx context.Context, templateID id.Zid) (*template.Template, error) {
	if t, ok := wui.cacheGetTemplate(templateID); ok {
		return t, nil
	}
	realTemplateZettel, err := wui.box.GetZettel(ctx, templateID)
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
	Lang              string
	MetaHeader        string
	CSSBaseURL        string
	CSSUserURL        string
	CSSRoleURL        string
	Title             string
	HomeURL           string
	WithUser          bool
	WithAuth          bool
	UserIsValid       bool
	UserZettelURL     string
	UserIdent         string
	LoginURL          string
	LogoutURL         string
	ListZettelURL     string
	ListRolesURL      string
	ListTagsURL       string
	CanRefresh        bool
	RefreshURL        string
	HasNewZettelLinks bool
	NewZettelLinks    []simpleLink
	SearchURL         string
	QueryKeySearch    string
	Content           string
	FooterHTML        string
}

func (wui *WebUI) makeBaseData(ctx context.Context, lang, title, roleCSSURL string, user *meta.Meta, data *baseData) {
	var userZettelURL string
	var userIdent string

	userIsValid := user != nil
	if userIsValid {
		userZettelURL = wui.NewURLBuilder('h').SetZid(api.ZettelID(user.Zid.String())).String()
		userIdent = user.GetDefault(api.KeyUserID, "")
	}
	newZettelLinks := wui.fetchNewTemplates(ctx, user)

	data.Lang = lang
	data.CSSBaseURL = wui.cssBaseURL
	data.CSSUserURL = wui.cssUserURL
	data.CSSRoleURL = roleCSSURL
	data.Title = title
	data.HomeURL = wui.homeURL
	data.WithAuth = wui.withAuth
	data.WithUser = data.WithAuth
	data.UserIsValid = userIsValid
	data.UserZettelURL = userZettelURL
	data.UserIdent = userIdent
	data.LoginURL = wui.loginURL
	data.LogoutURL = wui.logoutURL
	data.ListZettelURL = wui.listZettelURL
	data.ListRolesURL = wui.listRolesURL
	data.ListTagsURL = wui.listTagsURL
	data.CanRefresh = wui.canRefresh(user)
	data.RefreshURL = wui.refreshURL
	data.HasNewZettelLinks = len(newZettelLinks) > 0
	data.NewZettelLinks = newZettelLinks
	data.SearchURL = wui.searchURL
	data.QueryKeySearch = api.QueryKeySearch
	data.FooterHTML = wui.rtConfig.GetFooterHTML()
}

// htmlAttrNewWindow returns HTML attribute string for opening a link in a new window.
// If hasURL is false an empty string is returned.
func htmlAttrNewWindow(hasURL bool) string {
	if hasURL {
		return " target=\"_blank\" ref=\"noopener noreferrer\""
	}
	return ""
}

func (wui *WebUI) fetchNewTemplates(ctx context.Context, user *meta.Meta) (result []simpleLink) {
	ctx = box.NoEnrichContext(ctx)
	if !wui.canCreate(ctx, user) {
		return nil
	}
	menu, err := wui.box.GetZettel(ctx, id.TOCNewTemplateZid)
	if err != nil {
		return nil
	}
	refs := collect.Order(parser.ParseZettel(menu, "", wui.rtConfig))
	for _, ref := range refs {
		zid, err2 := id.Parse(ref.URL.Path)
		if err2 != nil {
			continue
		}
		m, err2 := wui.box.GetMeta(ctx, zid)
		if err2 != nil {
			continue
		}
		if !wui.policy.CanRead(user, m) {
			continue
		}
		title := config.GetTitle(m, wui.rtConfig)
		astTitle := parser.ParseMetadata(title)
		env := encoder.Environment{Lang: config.GetLang(m, wui.rtConfig)}
		menuTitle, err2 := encodeInlines(&astTitle, api.EncoderHTML, &env)
		if err2 != nil {
			menuTitle, err2 = encodeInlines(&astTitle, api.EncoderText, nil)
			if err2 != nil {
				menuTitle = title
			}
		}
		result = append(result, simpleLink{
			Text: menuTitle,
			URL: wui.NewURLBuilder('c').SetZid(api.ZettelID(m.Zid.String())).
				AppendQuery(queryKeyAction, valueActionNew).String(),
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
		wui.log.Error().Msg(err.Error())
	}
	user := wui.getUser(ctx)
	var base baseData
	wui.makeBaseData(ctx, api.ValueLangEN, "Error", "", user, &base)
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
		wui.log.IfErr(err).Zid(id.BaseTemplateZid).Msg("Unable to get template")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	t, err := wui.getTemplate(ctx, templateID)
	if err != nil {
		wui.log.IfErr(err).Zid(templateID).Msg("Unable to get template")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if user := wui.getUser(ctx); user != nil {
		if tok, err1 := wui.token.GetToken(user, wui.tokenLifetime, auth.KindHTML); err1 == nil {
			wui.setToken(w, tok)
		}
	}
	var content bytes.Buffer
	err = t.Render(&content, data)
	if err == nil {
		wui.prepareAndWriteHeader(w, code)
		err = writeHTMLStart(w, base.Lang)
		if err == nil {
			base.Content = content.String()
			err = bt.Render(w, base)
			if err == nil {
				err = wui.writeHTMLEnd(w)
			}
		}
	}
	if err != nil {
		wui.log.IfErr(err).Msg("Unable to write HTML via template")
	}
}

func writeHTMLStart(w http.ResponseWriter, lang string) error {
	_, err := io.WriteString(w, "<!DOCTYPE html>\n<html")
	if err != nil {
		return err
	}
	if lang != "" {
		_, err = io.WriteString(w, " lang=\"")
		if err == nil {
			_, err = io.WriteString(w, lang)
		}
		if err == nil {
			_, err = io.WriteString(w, "\">\n<head>\n")
		}
	} else {
		_, err = io.WriteString(w, ">\n<head>\n")
	}
	return err
}

func (wui *WebUI) writeHTMLEnd(w http.ResponseWriter) error {
	if wui.debug {
		_, err := io.WriteString(w, "<div><b>WARNING: Debug mode is enabled. DO NOT USE IN PRODUCTION!</b></div>\n")
		if err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "</body>\n</html>")
	return err
}

func (wui *WebUI) getUser(ctx context.Context) *meta.Meta { return wui.ab.GetUser(ctx) }

// GetURLPrefix returns the configured URL prefix of the web server.
func (wui *WebUI) GetURLPrefix() string { return wui.ab.GetURLPrefix() }

// NewURLBuilder creates a new URL builder object with the given key.
func (wui *WebUI) NewURLBuilder(key byte) *api.URLBuilder { return wui.ab.NewURLBuilder(key) }

func (wui *WebUI) clearToken(ctx context.Context, w http.ResponseWriter) context.Context {
	return wui.ab.ClearToken(ctx, w)
}

func (wui *WebUI) setToken(w http.ResponseWriter, token []byte) {
	wui.ab.SetToken(w, token, wui.tokenLifetime)
}

func (wui *WebUI) prepareAndWriteHeader(w http.ResponseWriter, statusCode int) {
	h := adapter.PrepareHeader(w, "text/html; charset=utf-8")
	h.Set("Content-Security-Policy", "default-src 'self'; img-src * data:; style-src 'self' 'unsafe-inline'")
	h.Set("Permissions-Policy", "payment=(), interest-cohort=()")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("X-Content-Type-Options", "nosniff")
	if !wui.debug {
		h.Set("X-Frame-Options", "sameorigin")
	}
	w.WriteHeader(statusCode)
}
