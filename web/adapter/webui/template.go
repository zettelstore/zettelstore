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
	"net/http"
	"net/url"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/sx.fossil"
	"zettelstore.de/sx.fossil/sxbuiltins"
	"zettelstore.de/sx.fossil/sxeval"
	"zettelstore.de/sx.fossil/sxhtml"
	"zettelstore.de/sx.fossil/sxreader"
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

func (wui *WebUI) createRenderEngine() *sxeval.Engine {
	root := sxeval.MakeRootEnvironment()
	engine := sxeval.MakeEngine(wui.sf, root)
	sxbuiltins.InstallQuoteSyntax(root, wui.symQuote)
	sxbuiltins.InstallQuasiQuoteSyntax(root, wui.symQQ, wui.symUQ, wui.symUQS)
	engine.BindSyntax("if", sxbuiltins.IfS)
	engine.BindSyntax("and", sxbuiltins.AndS)
	engine.BindSyntax("or", sxbuiltins.OrS)
	engine.BindSyntax("lambda", sxbuiltins.LambdaS)
	engine.BindSyntax("define", sxbuiltins.DefineS)
	engine.BindSyntax("let", sxbuiltins.LetS)
	engine.BindBuiltinFA("bound?", sxbuiltins.BoundP)
	engine.BindBuiltinFA("map", sxbuiltins.Map)
	engine.BindBuiltinFA("apply", sxbuiltins.Apply)
	engine.BindBuiltinA("list", sxbuiltins.List)
	engine.BindBuiltinA("append", sxbuiltins.Append)
	engine.BindBuiltinA("car", sxbuiltins.Car)
	engine.BindBuiltinA("cdr", sxbuiltins.Cdr)

	engine.BindBuiltinA("url-to-html", wui.url2html)
	return engine
}

func (wui *WebUI) url2html(args []sx.Object) (sx.Object, error) {
	err := sxbuiltins.CheckArgs(args, 1, 1)
	text, err := sxbuiltins.GetString(err, args, 0)
	if err != nil {
		return nil, err
	}
	if u, errURL := url.Parse(text.String()); errURL == nil {
		if us := u.String(); us != "" {
			return sx.MakeList(
				wui.symA,
				sx.MakeList(
					wui.symAttr,
					sx.Cons(wui.symHref, sx.MakeString(us)),
					sx.Cons(wui.sf.MustMake("target"), sx.MakeString("_blank")),
					sx.Cons(wui.sf.MustMake("rel"), sx.MakeString("noopener noreferrer")),
				),
				text), nil
		}
	}
	return text, nil
}

func (wui *WebUI) getParentEnv(ctx context.Context) sxeval.Environment {
	wui.mxZettelEnv.Lock()
	defer wui.mxZettelEnv.Unlock()
	if parentEnv := wui.zettelEnv; parentEnv != nil {
		return parentEnv
	}
	zettelEnv, err := wui.loadAllSxnCodeZettel(ctx)
	if err != nil {
		wui.log.IfErr(err).Msg("loading zettel sxn")
		return wui.engine.RootEnvironment()
	}
	wui.zettelEnv = zettelEnv
	return zettelEnv
}

// createRenderEnv creates a new environment and populates it with all relevant data for the base template.
func (wui *WebUI) createRenderEnv(ctx context.Context, name, lang, title string, user *meta.Meta) (sxeval.Environment, renderBinder) {
	userIsValid, userZettelURL, userIdent := wui.getUserRenderData(user)
	env := sxeval.MakeChildEnvironment(wui.getParentEnv(ctx), name, 128)
	rb := makeRenderBinder(wui.sf, env, nil)
	rb.bindString("lang", sx.MakeString(lang))
	rb.bindString("css-base-url", sx.MakeString(wui.cssBaseURL))
	rb.bindString("css-user-url", sx.MakeString(wui.cssUserURL))
	rb.bindString("css-role-url", sx.MakeString(""))
	rb.bindString("title", sx.MakeString(title))
	rb.bindString("home-url", sx.MakeString(wui.homeURL))
	rb.bindString("with-auth", sx.MakeBoolean(wui.withAuth))
	rb.bindString("user-is-valid", sx.MakeBoolean(userIsValid))
	rb.bindString("user-zettel-url", sx.MakeString(userZettelURL))
	rb.bindString("user-ident", sx.MakeString(userIdent))
	rb.bindString("login-url", sx.MakeString(wui.loginURL))
	rb.bindString("logout-url", sx.MakeString(wui.logoutURL))
	rb.bindString("list-zettel-url", sx.MakeString(wui.listZettelURL))
	rb.bindString("list-roles-url", sx.MakeString(wui.listRolesURL))
	rb.bindString("list-tags-url", sx.MakeString(wui.listTagsURL))
	if wui.canRefresh(user) {
		rb.bindString("refresh-url", sx.MakeString(wui.refreshURL))
	}
	rb.bindString("new-zettel-links", wui.fetchNewTemplatesSxn(ctx, user))
	rb.bindString("search-url", sx.MakeString(wui.searchURL))
	rb.bindString("query-key-query", sx.MakeString(api.QueryKeyQuery))
	rb.bindString("query-key-seed", sx.MakeString(api.QueryKeySeed))
	rb.bindString("FOOTER", wui.calculateFooterSxn(ctx)) // TODO: use real footer
	rb.bindString("debug-mode", sx.MakeBoolean(wui.debug))
	rb.bindSymbol(wui.symMetaHeader, sx.Nil())
	rb.bindSymbol(wui.symDetail, sx.Nil())
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
	make func(string) (*sx.Symbol, error)
	env  sxeval.Environment
}

func makeRenderBinder(sf sx.SymbolFactory, env sxeval.Environment, err error) renderBinder {
	return renderBinder{make: sf.Make, env: env, err: err}
}
func (rb *renderBinder) bindString(key string, obj sx.Object) {
	if rb.err == nil {
		sym, err := rb.make(key)
		if err == nil {
			rb.env, rb.err = rb.env.Bind(sym, obj)
			return
		}
		rb.err = err
	}
}
func (rb *renderBinder) bindSymbol(sym *sx.Symbol, obj sx.Object) {
	if rb.err == nil {
		rb.env, rb.err = rb.env.Bind(sym, obj)
	}
}
func (rb *renderBinder) bindKeyValue(key string, value string) {
	rb.bindString("meta-"+key, sx.MakeString(value))
	if kt := meta.Type(key); kt.IsSet {
		rb.bindString("set-meta-"+key, makeStringList(meta.ListFromValue(value)))
	}
}

func (wui *WebUI) bindCommonZettelData(ctx context.Context, rb *renderBinder, user, m *meta.Meta, content *zettel.Content) {
	strZid := m.Zid.String()
	apiZid := api.ZettelID(strZid)
	newURLBuilder := wui.NewURLBuilder

	rb.bindString("zid", sx.MakeString(strZid))
	rb.bindString("web-url", sx.MakeString(wui.NewURLBuilder('h').SetZid(apiZid).String()))
	if content != nil && wui.canWrite(ctx, user, m, *content) {
		rb.bindString("edit-url", sx.MakeString(newURLBuilder('e').SetZid(apiZid).String()))
	}
	rb.bindString("info-url", sx.MakeString(newURLBuilder('i').SetZid(apiZid).String()))
	if wui.canCreate(ctx, user) {
		if content != nil && !content.IsBinary() {
			rb.bindString("copy-url", sx.MakeString(wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionCopy).String()))
		}
		rb.bindString("version-url", sx.MakeString(wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionVersion).String()))
		rb.bindString("child-url", sx.MakeString(wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionChild).String()))
		rb.bindString("folge-url", sx.MakeString(wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionFolge).String()))
	}
	if wui.canRename(ctx, user, m) {
		rb.bindString("rename-url", sx.MakeString(wui.NewURLBuilder('b').SetZid(apiZid).String()))
	}
	if wui.canDelete(ctx, user, m) {
		rb.bindString("delete-url", sx.MakeString(wui.NewURLBuilder('d').SetZid(apiZid).String()))
	}
	if val, found := m.Get(api.KeyUselessFiles); found {
		rb.bindString("useless", sx.Cons(sx.MakeString(val), nil))
	}
	rb.bindString("context-url", sx.MakeString(wui.NewURLBuilder('h').AppendQuery(strZid+" "+api.ContextDirective).String()))

	// Ensure to have title, role, tags, and syntax included as "meta-*"
	rb.bindKeyValue(api.KeyTitle, m.GetDefault(api.KeyTitle, ""))
	rb.bindKeyValue(api.KeyRole, m.GetDefault(api.KeyRole, ""))
	rb.bindKeyValue(api.KeyTags, m.GetDefault(api.KeyTags, ""))
	rb.bindKeyValue(api.KeySyntax, m.GetDefault(api.KeySyntax, ""))
	sentinel := sx.Cons(nil, nil)
	curr := sentinel
	for _, p := range m.ComputedPairs() {
		key, value := p.Key, p.Value
		curr = curr.AppendBang(sx.Cons(sx.MakeString(key), sx.MakeString(value)))

		rb.bindKeyValue(key, value)
	}
	rb.bindString("metapairs", sentinel.Tail())
}

func (wui *WebUI) fetchNewTemplatesSxn(ctx context.Context, user *meta.Meta) (lst *sx.Pair) {
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
		text := sx.MakeString(parser.NormalizedSpacedText(z.Meta.GetTitle()))
		link := sx.MakeString(wui.NewURLBuilder('c').SetZid(api.ZettelID(zid.String())).
			AppendKVQuery(queryKeyAction, valueActionNew).String())

		lst = lst.Cons(sx.Cons(text, link))
	}
	return lst
}
func (wui *WebUI) calculateFooterSxn(ctx context.Context) *sx.Pair {
	if footerZid, err := id.Parse(wui.rtConfig.Get(ctx, nil, config.KeyFooterZettel)); err == nil {
		if zn, err2 := wui.evalZettel.Run(ctx, footerZid, ""); err2 == nil {
			htmlEnc := wui.getSimpleHTMLEncoder().SetUnique("footer-")
			if content, endnotes, err3 := htmlEnc.BlocksSxn(&zn.Ast); err3 == nil {
				if content != nil && endnotes != nil {
					content.LastPair().SetCdr(sx.Cons(endnotes, nil))
				}
				return content
			}
		}
	}
	return nil
}

func (wui *WebUI) getSxnTemplate(ctx context.Context, zid id.Zid, env sxeval.Environment) (sxeval.Expr, error) {
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

	wui.setSxnCache(zid, wui.engine.Rework(t))
	return t, nil
}
func (wui *WebUI) makeZettelReader(ctx context.Context, zid id.Zid) (*sxreader.Reader, error) {
	ztl, err := wui.box.GetZettel(ctx, zid)
	if err != nil {
		return nil, err
	}

	reader := sxreader.MakeReader(bytes.NewReader(ztl.Content.AsBytes()), sxreader.WithSymbolFactory(wui.sf))
	sxbuiltins.InstallQuoteReader(reader, wui.symQuote, '\'')
	sxbuiltins.InstallQuasiQuoteReader(reader, wui.symQQ, '`', wui.symUQ, ',', wui.symUQS, '@')
	return reader, nil
}

func (wui *WebUI) evalSxnTemplate(ctx context.Context, zid id.Zid, env sxeval.Environment) (sx.Object, error) {
	templateExpr, err := wui.getSxnTemplate(ctx, zid, env)
	if err != nil {
		return nil, err
	}
	return wui.engine.Execute(env, templateExpr)
}

func (wui *WebUI) renderSxnTemplate(ctx context.Context, w http.ResponseWriter, templateID id.Zid, env sxeval.Environment) error {
	return wui.renderSxnTemplateStatus(ctx, w, http.StatusOK, templateID, env)
}
func (wui *WebUI) renderSxnTemplateStatus(ctx context.Context, w http.ResponseWriter, code int, templateID id.Zid, env sxeval.Environment) error {
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
	rb.bindString("heading", sx.MakeString(http.StatusText(code)))
	rb.bindString("message", sx.MakeString(text))
	if rb.err == nil {
		rb.err = wui.renderSxnTemplate(ctx, w, id.ErrorTemplateZid, env)
	}
	if errBind := rb.err; errBind != nil {
		wui.log.Error().Err(errBind).Msg("while rendering error message")
		fmt.Fprintf(w, "Error while rendering error message: %v", errBind)
	}
}

func makeStringList(sl []string) *sx.Pair {
	if len(sl) == 0 {
		return nil
	}
	result := sx.Nil()
	for i := len(sl) - 1; i >= 0; i-- {
		result = result.Cons(sx.MakeString(sl[i]))
	}
	return result
}
