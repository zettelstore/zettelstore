//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
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

	"codeberg.org/t73fde/sxhtml"
	"codeberg.org/t73fde/sxpf"
	"codeberg.org/t73fde/sxpf/builtins/cond"
	"codeberg.org/t73fde/sxpf/builtins/quote"
	"codeberg.org/t73fde/sxpf/eval"
	"codeberg.org/t73fde/sxpf/reader"
	"zettelstore.de/c/api"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

func (wui *WebUI) createRenderEngine() *eval.Engine {
	root := sxpf.MakeRootEnvironment()
	engine := eval.MakeEngine(wui.sf, root, eval.MakeDefaultParser(), eval.MakeSimpleExecutor())
	quote.InstallQuoteSyntax(root, wui.symQuote)
	quote.InstallQuasiQuoteSyntax(root, wui.symQQ, wui.symUQ, wui.symUQS)
	engine.BindSyntax("if", cond.IfS)
	return engine
}

// createRenderEnv creates a new environment and populates it with all relevant data for the base template.
func (wui *WebUI) createRenderEnv(parent sxpf.Environment, name, lang, title string, user *meta.Meta) (sxpf.Environment, error) {
	userIsValid, userZettelURL, userIdent := wui.getUserRenderData(user)
	sf := wui.sf
	env := sxpf.MakeChildEnvironment(parent, name, 128)
	err := bindRenderEnv(nil, env, sf, "lang", sxpf.MakeString(lang))
	err = bindRenderEnv(err, env, sf, "css-base-url", sxpf.MakeString(wui.cssBaseURL))
	err = bindRenderEnv(err, env, sf, "css-user-url", sxpf.MakeString(wui.cssUserURL))
	err = bindRenderEnv(err, env, sf, "css-role-url", sxpf.MakeString(""))
	err = bindRenderEnv(err, env, sf, "title", sxpf.MakeString(title))
	err = bindRenderEnv(err, env, sf, "home-url", sxpf.MakeString(wui.homeURL))
	err = bindRenderEnv(err, env, sf, "with-auth", sxpf.MakeBoolean(wui.withAuth))
	err = bindRenderEnv(err, env, sf, "user-is-valid", sxpf.MakeBoolean(userIsValid))
	err = bindRenderEnv(err, env, sf, "user-zettel-url", sxpf.MakeString(userZettelURL))
	err = bindRenderEnv(err, env, sf, "user-ident", sxpf.MakeString(userIdent))
	err = bindRenderEnv(err, env, sf, "login-url", sxpf.MakeString(wui.loginURL))
	err = bindRenderEnv(err, env, sf, "logout-url", sxpf.MakeString(wui.logoutURL))
	err = bindRenderEnv(err, env, sf, "list-zettel-url", sxpf.MakeString(wui.listZettelURL))
	err = bindRenderEnv(err, env, sf, "list-roles-url", sxpf.MakeString(wui.listRolesURL))
	err = bindRenderEnv(err, env, sf, "list-tags-url", sxpf.MakeString(wui.listTagsURL))
	err = bindRenderEnv(err, env, sf, "can-refresh", sxpf.MakeBoolean(wui.canRefresh(user)))
	err = bindRenderEnv(err, env, sf, "refresh-url", sxpf.MakeString(wui.refreshURL))

	// data.NewZettelLinks = createSimpleLinks(wui.fetchNewTemplates(ctx, user))

	err = bindRenderEnv(err, env, sf, "search-url", sxpf.MakeString(wui.searchURL))
	err = bindRenderEnv(err, env, sf, "query-key-query", sxpf.MakeString(api.QueryKeyQuery))
	err = bindRenderEnv(err, env, sf, "query-key-seed", sxpf.MakeString(api.QueryKeySeed))

	// data.FooterHTML = wui.calculateFooterHTML(ctx)
	err = bindRenderEnv(err, env, sf, "FOOTER", sxpf.Nil()) // TODO: use real footer

	err = bindRenderEnv(err, env, sf, "debug-mode", sxpf.MakeBoolean(wui.debug))
	if err == nil {
		err = env.Bind(wui.symMetaHeader, sxpf.Nil())
	}
	if err == nil {
		err = env.Bind(wui.symDetail, sxpf.Nil())
	}

	return env, err
}

func (wui *WebUI) getUserRenderData(user *meta.Meta) (bool, string, string) {
	if user == nil {
		return false, "", ""
	}
	return true, wui.NewURLBuilder('h').SetZid(api.ZettelID(user.Zid.String())).String(), user.GetDefault(api.KeyUserID, "")
}

func bindRenderEnv(err error, env sxpf.Environment, sf sxpf.SymbolFactory, key string, obj sxpf.Object) error {
	if err != nil {
		return err
	}
	sym, err := sf.Make(key)
	if err != nil {
		return err
	}
	return env.Bind(sym, obj)
}

func (wui *WebUI) getSxnTemplate(ctx context.Context, zid id.Zid) (*sxpf.List, error) {
	wui.mxCache.RLock()
	t, ok := wui.templateSxnCache[zid]
	wui.mxCache.RUnlock()
	if ok {
		return t, nil
	}

	templateZettel, err := wui.box.GetZettel(ctx, zid)
	if err != nil {
		return nil, err
	}

	reader := reader.MakeReader(bytes.NewReader(templateZettel.Content.AsBytes()), reader.WithSymbolFactory(wui.sf))
	quote.InstallQuoteReader(reader, wui.symQuote, '\'')
	quote.InstallQuasiQuoteReader(reader, wui.symQQ, '`', wui.symUQ, ',', wui.symUQS, '@')

	obj, err := reader.Read()
	if err != nil {
		return nil, err
	}
	t = sxpf.MakeList(wui.symQQ, obj)

	wui.mxCache.Lock()
	wui.templateSxnCache[zid] = t
	wui.mxCache.Unlock()
	return t, nil
}

func (wui *WebUI) evalSxnTemplate(ctx context.Context, zid id.Zid, env sxpf.Environment) (sxpf.Object, error) {
	templateObj, err := wui.getSxnTemplate(ctx, zid)
	if err != nil {
		return nil, err
	}
	return wui.engine.Eval(env, templateObj)
}

func (wui *WebUI) renderSxnTemplate(ctx context.Context, w http.ResponseWriter, templateID id.Zid, env sxpf.Environment) error {
	return wui.renderSxnTemplateStatus(ctx, w, http.StatusOK, templateID, env)
}
func (wui *WebUI) renderSxnTemplateStatus(ctx context.Context, w http.ResponseWriter, code int, templateID id.Zid, env sxpf.Environment) error {
	detailObj, err := wui.evalSxnTemplate(ctx, templateID, env)
	if err != nil {
		return err
	}
	env.Bind(wui.symDetail, detailObj)

	pageObj, err := wui.evalSxnTemplate(ctx, id.BaseTemplateZid+30000, env)
	if err != nil {
		return err
	}

	gen := sxhtml.NewGenerator(wui.sf, sxhtml.WithNewline)
	var sb bytes.Buffer
	_, err = gen.WriteHTML(&sb, pageObj)
	if err != nil {
		return err
	}
	wui.prepareAndWriteHeader(w, code)
	_, err = w.Write(sb.Bytes())
	wui.log.IfErr(err).Msg("Unable to write HTML via template")
	return err
}
