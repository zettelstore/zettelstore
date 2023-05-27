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
	"codeberg.org/t73fde/sxpf/builtins"
	"codeberg.org/t73fde/sxpf/builtins/binding"
	"codeberg.org/t73fde/sxpf/builtins/boolean"
	"codeberg.org/t73fde/sxpf/builtins/callable"
	"codeberg.org/t73fde/sxpf/builtins/cond"
	"codeberg.org/t73fde/sxpf/builtins/env"
	"codeberg.org/t73fde/sxpf/builtins/list"
	"codeberg.org/t73fde/sxpf/builtins/quote"
	"codeberg.org/t73fde/sxpf/eval"
	"codeberg.org/t73fde/sxpf/reader"
	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/collect"
	"zettelstore.de/z/config"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

func (wui *WebUI) createRenderEngine() *eval.Engine {
	root := sxpf.MakeRootEnvironment()
	engine := eval.MakeEngine(wui.sf, root, eval.MakeDefaultParser(), eval.MakeSimpleExecutor())
	quote.InstallQuoteSyntax(root, wui.symQuote)
	quote.InstallQuasiQuoteSyntax(root, wui.symQQ, wui.symUQ, wui.symUQS)
	engine.BindSyntax("if", cond.IfS)
	engine.BindSyntax("and", boolean.AndS)
	engine.BindSyntax("or", boolean.OrS)
	engine.BindSyntax("let", binding.LetS)
	engine.BindBuiltinEEA("bound?", env.BoundP)
	engine.BindBuiltinEEA("map", callable.Map)
	engine.BindBuiltinA("list", list.List)
	engine.BindBuiltinA("pair-to-href", wui.sxnPairToHref)
	engine.BindBuiltinA("pair-to-href-li", wui.sxnPairToHrefLi)
	return engine
}

func (wui *WebUI) sxnPairToHref(args []sxpf.Object) (sxpf.Object, error) {
	err := builtins.CheckArgs(args, 1, 1)
	pair, err := builtins.GetList(err, args, 0)
	if err != nil {
		return nil, err
	}
	href := sxpf.MakeList(
		wui.symA,
		sxpf.MakeList(wui.symAttr, sxpf.Cons(wui.symHref, pair.Cdr())),
		pair.Car(),
	)
	return href, nil
}
func (wui *WebUI) sxnPairToHrefLi(args []sxpf.Object) (sxpf.Object, error) {
	href, err := wui.sxnPairToHref(args)
	if err != nil {
		return nil, err
	}
	return sxpf.MakeList(wui.symLi, href), nil
}

// createRenderEnv creates a new environment and populates it with all relevant data for the base template.
func (wui *WebUI) createRenderEnv(ctx context.Context, name, lang, title string, user *meta.Meta) (sxpf.Environment, error) {
	userIsValid, userZettelURL, userIdent := wui.getUserRenderData(user)
	env := sxpf.MakeChildEnvironment(wui.engine.RootEnvironment(), name, 128)
	rb := makeRenderBinder(wui.sf, env, nil)
	rb.bindString("lang", sxpf.MakeString(lang))
	rb.bindString("css-base-url", sxpf.MakeString(wui.cssBaseURL))
	rb.bindString("css-user-url", sxpf.MakeString(wui.cssUserURL))
	rb.bindString("css-role-url", sxpf.MakeString(""))
	rb.bindString("title", sxpf.MakeString(title))
	rb.bindString("home-url", sxpf.MakeString(wui.homeURL))
	rb.bindString("with-auth", sxpf.MakeBoolean(wui.withAuth))
	rb.bindString("user-is-valid", sxpf.MakeBoolean(userIsValid))
	rb.bindString("user-zettel-url", sxpf.MakeString(userZettelURL))
	rb.bindString("user-ident", sxpf.MakeString(userIdent))
	rb.bindString("login-url", sxpf.MakeString(wui.loginURL))
	rb.bindString("logout-url", sxpf.MakeString(wui.logoutURL))
	rb.bindString("list-zettel-url", sxpf.MakeString(wui.listZettelURL))
	rb.bindString("list-roles-url", sxpf.MakeString(wui.listRolesURL))
	rb.bindString("list-tags-url", sxpf.MakeString(wui.listTagsURL))
	rb.bindString("can-refresh", sxpf.MakeBoolean(wui.canRefresh(user)))
	rb.bindString("refresh-url", sxpf.MakeString(wui.refreshURL))
	rb.bindString("new-zettel-links", wui.fetchNewTemplatesSxn(ctx, user))
	rb.bindString("search-url", sxpf.MakeString(wui.searchURL))
	rb.bindString("query-key-query", sxpf.MakeString(api.QueryKeyQuery))
	rb.bindString("query-key-seed", sxpf.MakeString(api.QueryKeySeed))
	rb.bindString("FOOTER", wui.calculateFooterSxn(ctx)) // TODO: use real footer
	rb.bindString("debug-mode", sxpf.MakeBoolean(wui.debug))
	rb.bindSymbol(wui.symMetaHeader, sxpf.Nil())
	rb.bindSymbol(wui.symDetail, sxpf.Nil())
	return env, rb.err
}

func (wui *WebUI) getUserRenderData(user *meta.Meta) (bool, string, string) {
	if user == nil {
		return false, "", ""
	}
	return true, wui.NewURLBuilder('h').SetZid(api.ZettelID(user.Zid.String())).String(), user.GetDefault(api.KeyUserID, "")
}

type renderBinder struct {
	err  error
	make func(string) (*sxpf.Symbol, error)
	bind func(*sxpf.Symbol, sxpf.Object) error
}

func makeRenderBinder(sf sxpf.SymbolFactory, env sxpf.Environment, err error) renderBinder {
	return renderBinder{make: sf.Make, bind: env.Bind, err: err}
}
func (rb *renderBinder) bindString(key string, obj sxpf.Object) {
	if rb.err == nil {
		sym, err := rb.make(key)
		if err == nil {
			rb.err = rb.bind(sym, obj)
			return
		}
		rb.err = err
	}
}
func (rb *renderBinder) bindSymbol(sym *sxpf.Symbol, obj sxpf.Object) {
	if rb.err == nil {
		rb.err = rb.bind(sym, obj)
	}
}

func (wui *WebUI) fetchNewTemplatesSxn(ctx context.Context, user *meta.Meta) (lst *sxpf.List) {
	if !wui.canCreate(ctx, user) {
		return nil
	}
	ctx = box.NoEnrichContext(ctx)
	menu, err := wui.box.GetZettel(ctx, id.TOCNewTemplateZid)
	if err != nil {
		return nil
	}
	refs := collect.Order(parser.ParseZettel(ctx, menu, "", wui.rtConfig))
	for i := len(refs) - 1; i >= 0; i-- {
		zid, err2 := id.Parse(refs[i].URL.Path)
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
		text := sxpf.MakeString(parser.NormalizedSpacedText(m.GetTitle()))
		link := sxpf.MakeString(wui.NewURLBuilder('c').SetZid(api.ZettelID(m.Zid.String())).
			AppendKVQuery(queryKeyAction, valueActionNew).String())

		lst = lst.Cons(sxpf.Cons(text, link))
	}
	return lst
}
func (wui *WebUI) calculateFooterSxn(ctx context.Context) *sxpf.List {
	if footerZid, err := id.Parse(wui.rtConfig.Get(ctx, nil, config.KeyFooterZettel)); err == nil {
		if zn, err2 := wui.evalZettel.Run(ctx, footerZid, ""); err2 == nil {
			htmlEnc := wui.getSimpleHTMLEncoder().SetUnique("footer-")
			if content, endnotes, err3 := htmlEnc.BlocksSxn(&zn.Ast); err3 == nil {
				if content != nil && endnotes != nil {
					content.LastPair().SetCdr(sxpf.Cons(endnotes, nil))
				}
				return content
			}
		}
	}
	return nil
}

func (wui *WebUI) getSxnTemplate(ctx context.Context, zid id.Zid, env sxpf.Environment) (eval.Expr, error) {
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
	form := sxpf.MakeList(wui.symQQ, obj)
	t, err = wui.engine.Parse(env, form)
	if err != nil {
		return nil, err
	}

	wui.mxCache.Lock()
	wui.templateSxnCache[zid] = t
	wui.mxCache.Unlock()
	return t, nil
}

func (wui *WebUI) evalSxnTemplate(ctx context.Context, zid id.Zid, env sxpf.Environment) (sxpf.Object, error) {
	templateExpr, err := wui.getSxnTemplate(ctx, zid, env)
	if err != nil {
		return nil, err
	}
	return wui.engine.Execute(env, templateExpr)
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

	pageObj, err := wui.evalSxnTemplate(ctx, id.BaseTemplateZid, env)
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
