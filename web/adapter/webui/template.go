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
	"fmt"
	"io"
	"net/http"
	"net/url"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/sx.fossil/sxhtml"
	"zettelstore.de/sx.fossil/sxpf"
	"zettelstore.de/sx.fossil/sxpf/builtins"
	"zettelstore.de/sx.fossil/sxpf/builtins/binding"
	"zettelstore.de/sx.fossil/sxpf/builtins/boolean"
	"zettelstore.de/sx.fossil/sxpf/builtins/callable"
	"zettelstore.de/sx.fossil/sxpf/builtins/cond"
	"zettelstore.de/sx.fossil/sxpf/builtins/define"
	"zettelstore.de/sx.fossil/sxpf/builtins/env"
	"zettelstore.de/sx.fossil/sxpf/builtins/list"
	"zettelstore.de/sx.fossil/sxpf/builtins/quote"
	"zettelstore.de/sx.fossil/sxpf/eval"
	"zettelstore.de/sx.fossil/sxpf/reader"
	"zettelstore.de/z/box"
	"zettelstore.de/z/collect"
	"zettelstore.de/z/config"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/server"
	"zettelstore.de/z/zettel"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

func (wui *WebUI) createRenderEngine() *eval.Engine {
	root := sxpf.MakeRootEnvironment()
	engine := eval.MakeEngine(wui.sf, root)
	quote.InstallQuoteSyntax(root, wui.symQuote)
	quote.InstallQuasiQuoteSyntax(root, wui.symQQ, wui.symUQ, wui.symUQS)
	engine.BindSyntax("if", cond.IfS)
	engine.BindSyntax("and", boolean.AndS)
	engine.BindSyntax("or", boolean.OrS)
	engine.BindSyntax("lambda", callable.LambdaS)
	engine.BindSyntax("define", define.DefineS)
	engine.BindSyntax("let", binding.LetS)
	engine.BindBuiltinEEA("bound?", env.BoundP)
	engine.BindBuiltinEEA("map", callable.Map)
	engine.BindBuiltinEEA("apply", callable.Apply)
	engine.BindBuiltinA("list", list.List)
	engine.BindBuiltinA("append", list.Append)
	engine.BindBuiltinA("car", list.Car)
	engine.BindBuiltinA("cdr", list.Cdr)

	engine.BindBuiltinA("url-to-html", wui.url2html)
	return engine
}

func (wui *WebUI) url2html(args []sxpf.Object) (sxpf.Object, error) {
	err := builtins.CheckArgs(args, 1, 1)
	text, err := builtins.GetString(err, args, 0)
	if err != nil {
		return nil, err
	}
	if u, errURL := url.Parse(text.String()); errURL == nil {
		if us := u.String(); us != "" {
			return sxpf.MakeList(
				wui.symA,
				sxpf.MakeList(
					wui.symAttr,
					sxpf.Cons(wui.symHref, sxpf.MakeString(us)),
					sxpf.Cons(wui.sf.MustMake("target"), sxpf.MakeString("_blank")),
					sxpf.Cons(wui.sf.MustMake("rel"), sxpf.MakeString("noopener noreferrer")),
				),
				text), nil
		}
	}
	return text, nil
}

// createRenderEnv creates a new environment and populates it with all relevant data for the base template.
func (wui *WebUI) createRenderEnv(ctx context.Context, name, lang, title string, user *meta.Meta) (sxpf.Environment, renderBinder) {
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
	if wui.canRefresh(user) {
		rb.bindString("refresh-url", sxpf.MakeString(wui.refreshURL))
	}
	rb.bindString("new-zettel-links", wui.fetchNewTemplatesSxn(ctx, user))
	rb.bindString("search-url", sxpf.MakeString(wui.searchURL))
	rb.bindString("query-key-query", sxpf.MakeString(api.QueryKeyQuery))
	rb.bindString("query-key-seed", sxpf.MakeString(api.QueryKeySeed))
	rb.bindString("FOOTER", wui.calculateFooterSxn(ctx)) // TODO: use real footer
	rb.bindString("debug-mode", sxpf.MakeBoolean(wui.debug))
	rb.bindSymbol(wui.symMetaHeader, sxpf.Nil())
	rb.bindSymbol(wui.symDetail, sxpf.Nil())
	return env, rb
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
func (rb *renderBinder) bindKeyValue(key string, value string) {
	rb.bindString("meta-"+key, sxpf.MakeString(value))
	if kt := meta.Type(key); kt.IsSet {
		rb.bindString("set-meta-"+key, makeStringList(meta.ListFromValue(value)))
	}
}

func (wui *WebUI) bindCommonZettelData(ctx context.Context, rb *renderBinder, user, m *meta.Meta, content *zettel.Content) {
	strZid := m.Zid.String()
	apiZid := api.ZettelID(strZid)
	newURLBuilder := wui.NewURLBuilder

	rb.bindString("zid", sxpf.MakeString(strZid))
	rb.bindString("web-url", sxpf.MakeString(wui.NewURLBuilder('h').SetZid(apiZid).String()))
	if content != nil && wui.canWrite(ctx, user, m, *content) {
		rb.bindString("edit-url", sxpf.MakeString(newURLBuilder('e').SetZid(apiZid).String()))
	}
	rb.bindString("info-url", sxpf.MakeString(newURLBuilder('i').SetZid(apiZid).String()))
	if wui.canCreate(ctx, user) {
		if content != nil && !content.IsBinary() {
			rb.bindString("copy-url", sxpf.MakeString(wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionCopy).String()))
		}
		rb.bindString("version-url", sxpf.MakeString(wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionVersion).String()))
		rb.bindString("child-url", sxpf.MakeString(wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionChild).String()))
		rb.bindString("folge-url", sxpf.MakeString(wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionFolge).String()))
	}
	if wui.canRename(ctx, user, m) {
		rb.bindString("rename-url", sxpf.MakeString(wui.NewURLBuilder('b').SetZid(apiZid).String()))
	}
	if wui.canDelete(ctx, user, m) {
		rb.bindString("delete-url", sxpf.MakeString(wui.NewURLBuilder('d').SetZid(apiZid).String()))
	}
	if val, found := m.Get(api.KeyUselessFiles); found {
		rb.bindString("useless", sxpf.Cons(sxpf.MakeString(val), nil))
	}
	rb.bindString("context-url", sxpf.MakeString(wui.NewURLBuilder('h').AppendQuery(strZid+" "+api.ContextDirective).String()))

	// Ensure to have title, role, tags, and syntax included as "meta-*"
	rb.bindKeyValue(api.KeyTitle, m.GetDefault(api.KeyTitle, ""))
	rb.bindKeyValue(api.KeyRole, m.GetDefault(api.KeyRole, ""))
	rb.bindKeyValue(api.KeyTags, m.GetDefault(api.KeyTags, ""))
	rb.bindKeyValue(api.KeySyntax, m.GetDefault(api.KeySyntax, ""))
	sentinel := sxpf.Cons(nil, nil)
	curr := sentinel
	for _, p := range m.ComputedPairs() {
		key, value := p.Key, p.Value
		curr = curr.AppendBang(sxpf.Cons(sxpf.MakeString(key), sxpf.MakeString(value)))

		rb.bindKeyValue(key, value)
	}
	rb.bindString("metapairs", sentinel.Tail())
}

func (wui *WebUI) fetchNewTemplatesSxn(ctx context.Context, user *meta.Meta) (lst *sxpf.Pair) {
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
		z, err2 := wui.box.GetZettel(ctx, zid)
		if err2 != nil {
			continue
		}
		if !wui.policy.CanRead(user, z.Meta) {
			continue
		}
		text := sxpf.MakeString(parser.NormalizedSpacedText(z.Meta.GetTitle()))
		link := sxpf.MakeString(wui.NewURLBuilder('c').SetZid(api.ZettelID(zid.String())).
			AppendKVQuery(queryKeyAction, valueActionNew).String())

		lst = lst.Cons(sxpf.Cons(text, link))
	}
	return lst
}
func (wui *WebUI) calculateFooterSxn(ctx context.Context) *sxpf.Pair {
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

func (wui *WebUI) loadSxnCodeZettel(ctx context.Context, zid id.Zid) error {
	if expr := wui.getSxnCache(zid); expr != nil {
		return nil
	}
	rdr, err := wui.makeZettelReader(ctx, zid)
	if err != nil {
		return err
	}
	for {
		form, err2 := rdr.Read()
		if err2 != nil {
			if err2 == io.EOF {
				wui.setSxnCache(zid, eval.TrueExpr) // Hack to load only once
				return nil
			}
			return err2
		}
		wui.log.Trace().Str("form", form.Repr()).Msg("Load sxn code")

		_, err2 = wui.engine.Eval(wui.engine.GetToplevelEnv(), form)
		if err2 != nil {
			return err2
		}
	}
}

func (wui *WebUI) getSxnTemplate(ctx context.Context, zid id.Zid, env sxpf.Environment) (eval.Expr, error) {
	if t := wui.getSxnCache(zid); t != nil {
		return t, nil
	}

	reader, err := wui.makeZettelReader(ctx, zid)
	if err != nil {
		return nil, err
	}

	objs, err := reader.ReadAll()
	if err != nil {
		wui.log.IfErr(err).Zid(zid).Msg("reading sxn template")
		return nil, err
	}
	if len(objs) != 1 {
		return nil, fmt.Errorf("expected 1 expression in template, but got %d", len(objs))
	}
	t, err := wui.engine.Parse(env, objs[0])
	if err != nil {
		return nil, err
	}

	wui.setSxnCache(zid, wui.engine.Rework(env, t))
	return t, nil
}
func (wui *WebUI) makeZettelReader(ctx context.Context, zid id.Zid) (*reader.Reader, error) {
	ztl, err := wui.box.GetZettel(ctx, zid)
	if err != nil {
		return nil, err
	}

	reader := reader.MakeReader(bytes.NewReader(ztl.Content.AsBytes()), reader.WithSymbolFactory(wui.sf))
	quote.InstallQuoteReader(reader, wui.symQuote, '\'')
	quote.InstallQuasiQuoteReader(reader, wui.symQQ, '`', wui.symUQ, ',', wui.symUQS, '@')
	return reader, nil
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
	err := wui.loadSxnCodeZettel(ctx, id.TemplateSxnZid)
	if err != nil {
		return err
	}
	detailObj, err := wui.evalSxnTemplate(ctx, templateID, env)
	if err != nil {
		return err
	}
	env.Bind(wui.symDetail, detailObj)

	pageObj, err := wui.evalSxnTemplate(ctx, id.BaseTemplateZid, env)
	if err != nil {
		return err
	}
	wui.log.Debug().Str("page", pageObj.Repr()).Msg("render")

	gen := sxhtml.NewGenerator(wui.sf, sxhtml.WithNewline)
	var sb bytes.Buffer
	_, err = gen.WriteHTML(&sb, pageObj)
	if err != nil {
		return err
	}
	wui.prepareAndWriteHeader(w, code)
	_, err = w.Write(sb.Bytes())
	wui.log.IfErr(err).Msg("Unable to write HTML via template")
	return nil // No error reporting, since we do not know what happended during write to client.
}

func (wui *WebUI) reportError(ctx context.Context, w http.ResponseWriter, err error) {
	code, text := adapter.CodeMessageFromError(err)
	if code == http.StatusInternalServerError {
		wui.log.Error().Msg(err.Error())
	} else {
		wui.log.Trace().Err(err).Msg("reportError")
	}
	user := server.GetUser(ctx)
	env, rb := wui.createRenderEnv(ctx, "error", api.ValueLangEN, "Error", user)
	rb.bindString("heading", sxpf.MakeString(http.StatusText(code)))
	rb.bindString("message", sxpf.MakeString(text))
	if rb.err == nil {
		rb.err = wui.renderSxnTemplate(ctx, w, id.ErrorTemplateZid, env)
	}
	if errBind := rb.err; errBind != nil {
		wui.log.Error().Err(errBind).Msg("while rendering error message")
		fmt.Fprintf(w, "Error while rendering error message: %v", errBind)
	}
}

func makeStringList(sl []string) *sxpf.Pair {
	if len(sl) == 0 {
		return nil
	}
	result := sxpf.Nil()
	for i := len(sl) - 1; i >= 0; i-- {
		result = result.Cons(sxpf.MakeString(sl[i]))
	}
	return result
}