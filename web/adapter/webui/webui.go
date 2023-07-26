//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
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
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/sx.fossil/sxhtml"
	"zettelstore.de/sx.fossil/sxpf"
	"zettelstore.de/sx.fossil/sxpf/eval"
	"zettelstore.de/z/auth"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/logger"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/server"
	"zettelstore.de/z/zettel"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
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

	evalZettel *usecase.Evaluate

	mxCache       sync.RWMutex
	templateCache map[id.Zid]eval.Expr

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
	createNewURL  string

	sf      sxpf.SymbolFactory
	engine  *eval.Engine
	genHTML *sxhtml.Generator

	symQuote, symQQ *sxpf.Symbol
	symUQ, symUQS   *sxpf.Symbol
	symMetaHeader   *sxpf.Symbol
	symDetail       *sxpf.Symbol
	symA, symHref   *sxpf.Symbol
	symSpan         *sxpf.Symbol
	symAttr         *sxpf.Symbol
}

type webuiBox interface {
	CanCreateZettel(ctx context.Context) bool
	GetZettel(ctx context.Context, zid id.Zid) (zettel.Zettel, error)
	CanUpdateZettel(ctx context.Context, zettel zettel.Zettel) bool
	AllowRenameZettel(ctx context.Context, zid id.Zid) bool
	CanDeleteZettel(ctx context.Context, zid id.Zid) bool
}

// New creates a new WebUI struct.
func New(log *logger.Logger, ab server.AuthBuilder, authz auth.AuthzManager, rtConfig config.Config, token auth.TokenManager,
	mgr box.Manager, pol auth.Policy, evalZettel *usecase.Evaluate) *WebUI {
	loginoutBase := ab.NewURLBuilder('i')
	sf := sxpf.MakeMappedFactory()

	wui := &WebUI{
		log:      log,
		debug:    kernel.Main.GetConfig(kernel.CoreService, kernel.CoreDebug).(bool),
		ab:       ab,
		rtConfig: rtConfig,
		authz:    authz,
		token:    token,
		box:      mgr,
		policy:   pol,

		evalZettel: evalZettel,

		tokenLifetime: kernel.Main.GetConfig(kernel.WebService, kernel.WebTokenLifetimeHTML).(time.Duration),
		cssBaseURL:    ab.NewURLBuilder('z').SetZid(api.ZidBaseCSS).String(),
		cssUserURL:    ab.NewURLBuilder('z').SetZid(api.ZidUserCSS).String(),
		homeURL:       ab.NewURLBuilder('/').String(),
		listZettelURL: ab.NewURLBuilder('h').String(),
		listRolesURL:  ab.NewURLBuilder('h').AppendQuery(api.ActionSeparator + api.KeyRole).String(),
		listTagsURL:   ab.NewURLBuilder('h').AppendQuery(api.ActionSeparator + api.KeyTags).String(),
		refreshURL:    ab.NewURLBuilder('g').AppendKVQuery("_c", "r").String(),
		withAuth:      authz.WithAuth(),
		loginURL:      loginoutBase.String(),
		logoutURL:     loginoutBase.AppendKVQuery("logout", "").String(),
		searchURL:     ab.NewURLBuilder('h').String(),
		createNewURL:  ab.NewURLBuilder('c').String(),

		sf:            sf,
		genHTML:       sxhtml.NewGenerator(sf, sxhtml.WithNewline),
		symQuote:      sf.MustMake("quote"),
		symQQ:         sf.MustMake("quasiquote"),
		symUQ:         sf.MustMake("unquote"),
		symUQS:        sf.MustMake("unquote-splicing"),
		symDetail:     sf.MustMake("DETAIL"),
		symMetaHeader: sf.MustMake("META-HEADER"),
		symA:          sf.MustMake("a"),
		symHref:       sf.MustMake("href"),
		symSpan:       sf.MustMake("span"),
		symAttr:       sf.MustMake(sxhtml.NameSymAttr),
	}
	wui.engine = wui.createRenderEngine()
	wui.observe(box.UpdateInfo{Box: mgr, Reason: box.OnReload, Zid: id.Invalid})
	mgr.RegisterObserver(wui.observe)
	return wui
}

func (wui *WebUI) observe(ci box.UpdateInfo) {
	wui.mxCache.Lock()
	if ci.Reason == box.OnReload {
		wui.templateCache = make(map[id.Zid]eval.Expr, len(wui.templateCache))
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

func (wui *WebUI) setSxnCache(zid id.Zid, expr eval.Expr) {
	wui.mxCache.Lock()
	wui.templateCache[zid] = expr
	wui.mxCache.Unlock()
}
func (wui *WebUI) getSxnCache(zid id.Zid) eval.Expr {
	wui.mxCache.RLock()
	expr, found := wui.templateCache[zid]
	wui.mxCache.RUnlock()
	if found {
		return expr
	}
	return nil
}

func (wui *WebUI) retrieveCSSZidFromRole(ctx context.Context, m *meta.Meta) (id.Zid, error) {
	wui.mxRoleCSSMap.RLock()
	if wui.roleCSSMap == nil {
		wui.mxRoleCSSMap.RUnlock()
		wui.mxRoleCSSMap.Lock()
		zMap, err := wui.box.GetZettel(ctx, id.RoleCSSMapZid)
		if err == nil {
			wui.roleCSSMap = createRoleCSSMap(zMap.Meta)
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

func createRoleCSSMap(mMap *meta.Meta) map[string]id.Zid {
	result := make(map[string]id.Zid)
	for _, p := range mMap.PairsRest() {
		key := p.Key
		if len(key) < 9 || !strings.HasPrefix(key, "css-") || !strings.HasSuffix(key, "-zid") {
			continue
		}
		zid, err2 := id.Parse(p.Value)
		if err2 != nil {
			continue
		}
		result[key[4:len(key)-4]] = zid
	}
	return result
}

func (wui *WebUI) canCreate(ctx context.Context, user *meta.Meta) bool {
	m := meta.New(id.Invalid)
	return wui.policy.CanCreate(user, m) && wui.box.CanCreateZettel(ctx)
}

func (wui *WebUI) canWrite(
	ctx context.Context, user, meta *meta.Meta, content zettel.Content) bool {
	return wui.policy.CanWrite(user, meta, meta) &&
		wui.box.CanUpdateZettel(ctx, zettel.Zettel{Meta: meta, Content: content})
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

func (wui *WebUI) getSimpleHTMLEncoder() *htmlGenerator { return wui.createGenerator(wui) }

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
