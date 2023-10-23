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
	root := sxeval.MakeRootEnvironment(len(syntaxes) + len(builtins) + 3)
	engine := sxeval.MakeEngine(wui.sf, root)
	for _, syntax := range syntaxes {
		engine.BindSyntax(syntax)
	}
	for _, b := range builtins {
		engine.BindBuiltin(b)
	}
	engine.BindBuiltin(&sxeval.Builtin{
		Name:     "url-to-html",
		MinArity: 1,
		MaxArity: 1,
		IsPure:   true,
		Fn: func(_ *sxeval.Frame, args []sx.Object) (sx.Object, error) {
			text, err := sxbuiltins.GetString(args, 0)
			if err != nil {
				return nil, err
			}
			return wui.url2html(text), nil
		},
	})
	engine.BindBuiltin(&sxeval.Builtin{
		Name:     "zid-content-path",
		MinArity: 1,
		MaxArity: 1,
		IsPure:   true,
		Fn: func(_ *sxeval.Frame, args []sx.Object) (sx.Object, error) {
			s, err := sxbuiltins.GetString(args, 0)
			if err != nil {
				return nil, err
			}
			zid, err := id.Parse(s.String())
			if err != nil {
				return nil, fmt.Errorf("parsing zettel identfier %q: %w", s, err)
			}
			ub := wui.NewURLBuilder('z').SetZid(api.ZettelID(zid.String()))
			return sx.String(ub.String()), nil
		},
	})
	engine.BindBuiltin(&sxeval.Builtin{
		Name:     "query->url",
		MinArity: 1,
		MaxArity: 1,
		IsPure:   true,
		Fn: func(_ *sxeval.Frame, args []sx.Object) (sx.Object, error) {
			qs, err := sxbuiltins.GetString(args, 0)
			if err != nil {
				return nil, err
			}
			u := wui.NewURLBuilder('h').AppendQuery(qs.String())
			return sx.String(u.String()), nil
		},
	})
	root.Freeze()
	return engine
}

var (
	syntaxes = []*sxeval.Syntax{
		&sxbuiltins.QuoteS, &sxbuiltins.QuasiquoteS, // quote, quasiquote
		&sxbuiltins.UnquoteS, &sxbuiltins.UnquoteSplicingS, // unquote, unquote-splicing
		&sxbuiltins.DefVarS, &sxbuiltins.DefConstS, // defvar, defconst
		&sxbuiltins.SetXS,                       // set!
		&sxbuiltins.DefineS,                     // define (DEPRECATED)
		&sxbuiltins.DefunS, &sxbuiltins.LambdaS, // defun, lambda
		&sxbuiltins.CondS,     // cond
		&sxbuiltins.IfS,       // if
		&sxbuiltins.DefMacroS, // defmacro
	}
	builtins = []*sxeval.Builtin{
		&sxbuiltins.Identical,            // ==
		&sxbuiltins.NullP,                // null?
		&sxbuiltins.PairP,                // pair?
		&sxbuiltins.Car, &sxbuiltins.Cdr, // car, cdr
		&sxbuiltins.Caar, &sxbuiltins.Cadr, &sxbuiltins.Cdar, &sxbuiltins.Cddr,
		&sxbuiltins.Caaar, &sxbuiltins.Caadr, &sxbuiltins.Cadar, &sxbuiltins.Caddr,
		&sxbuiltins.Cdaar, &sxbuiltins.Cdadr, &sxbuiltins.Cddar, &sxbuiltins.Cdddr,
		&sxbuiltins.List,         // list
		&sxbuiltins.Append,       // append
		&sxbuiltins.Assoc,        // assoc
		&sxbuiltins.Map,          // map
		&sxbuiltins.Apply,        // apply
		&sxbuiltins.StringAppend, // string-append
		&sxbuiltins.BoundP,       // bound?
		&sxbuiltins.Defined,      // defined?
		&sxbuiltins.CurrentEnv,   // current-environment
		&sxbuiltins.EnvLookup,    // environment-lookup
	}
)

func (wui *WebUI) url2html(text sx.String) sx.Object {
	if u, errURL := url.Parse(text.String()); errURL == nil {
		if us := u.String(); us != "" {
			return sx.MakeList(
				wui.symA,
				sx.MakeList(
					wui.symAttr,
					sx.Cons(wui.symHref, sx.String(us)),
					sx.Cons(wui.sf.MustMake("target"), sx.String("_blank")),
					sx.Cons(wui.sf.MustMake("rel"), sx.String("noopener noreferrer")),
				),
				text)
		}
	}
	return text
}

func (wui *WebUI) getParentEnv(ctx context.Context) (sxeval.Environment, error) {
	wui.mxZettelEnv.Lock()
	defer wui.mxZettelEnv.Unlock()
	if parentEnv := wui.zettelEnv; parentEnv != nil {
		return parentEnv, nil
	}
	dag, zettelEnv, err := wui.loadAllSxnCodeZettel(ctx)
	if err != nil {
		wui.log.Error().Err(err).Msg("loading zettel sxn")
		return nil, err
	}
	wui.dag = dag
	wui.zettelEnv = zettelEnv
	return zettelEnv, nil
}

// createRenderEnv creates a new environment and populates it with all relevant data for the base template.
func (wui *WebUI) createRenderEnv(ctx context.Context, name, lang, title string, user *meta.Meta) (sxeval.Environment, renderBinder) {
	userIsValid, userZettelURL, userIdent := wui.getUserRenderData(user)
	parentEnv, err := wui.getParentEnv(ctx)
	env := sxeval.MakeChildEnvironment(parentEnv, name, 128)
	rb := makeRenderBinder(wui.sf, env, err)
	rb.bindString("lang", sx.String(lang))
	rb.bindString("css-base-url", sx.String(wui.cssBaseURL))
	rb.bindString("css-user-url", sx.String(wui.cssUserURL))
	rb.bindString("title", sx.String(title))
	rb.bindString("home-url", sx.String(wui.homeURL))
	rb.bindString("with-auth", sx.MakeBoolean(wui.withAuth))
	rb.bindString("user-is-valid", sx.MakeBoolean(userIsValid))
	rb.bindString("user-zettel-url", sx.String(userZettelURL))
	rb.bindString("user-ident", sx.String(userIdent))
	rb.bindString("login-url", sx.String(wui.loginURL))
	rb.bindString("logout-url", sx.String(wui.logoutURL))
	rb.bindString("list-zettel-url", sx.String(wui.listZettelURL))
	rb.bindString("list-roles-url", sx.String(wui.listRolesURL))
	rb.bindString("list-tags-url", sx.String(wui.listTagsURL))
	if wui.canRefresh(user) {
		rb.bindString("refresh-url", sx.String(wui.refreshURL))
	}
	rb.bindString("new-zettel-links", wui.fetchNewTemplatesSxn(ctx, user))
	rb.bindString("search-url", sx.String(wui.searchURL))
	rb.bindString("query-key-query", sx.String(api.QueryKeyQuery))
	rb.bindString("query-key-seed", sx.String(api.QueryKeySeed))
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
			rb.err = rb.env.Bind(sym, obj)
			return
		}
		rb.err = err
	}
}
func (rb *renderBinder) bindSymbol(sym *sx.Symbol, obj sx.Object) {
	if rb.err == nil {
		rb.err = rb.env.Bind(sym, obj)
	}
}
func (rb *renderBinder) bindKeyValue(key string, value string) {
	rb.bindString("meta-"+key, sx.String(value))
	if kt := meta.Type(key); kt.IsSet {
		rb.bindString("set-meta-"+key, makeStringList(meta.ListFromValue(value)))
	}
}
func (rb *renderBinder) rebindResolved(key, defKey string) {
	if rb.err == nil {
		sym, err := rb.make(key)
		if err == nil {
			if obj, found := sxeval.Resolve(rb.env, sym); found {
				rb.bindString(defKey, obj)
				return
			}
			return
		}
		rb.err = err
	}
}

func (wui *WebUI) bindCommonZettelData(ctx context.Context, rb *renderBinder, user, m *meta.Meta, content *zettel.Content) {
	strZid := m.Zid.String()
	apiZid := api.ZettelID(strZid)
	newURLBuilder := wui.NewURLBuilder

	rb.bindString("zid", sx.String(strZid))
	rb.bindString("web-url", sx.String(newURLBuilder('h').SetZid(apiZid).String()))
	if content != nil && wui.canWrite(ctx, user, m, *content) {
		rb.bindString("edit-url", sx.String(newURLBuilder('e').SetZid(apiZid).String()))
	}
	rb.bindString("info-url", sx.String(newURLBuilder('i').SetZid(apiZid).String()))
	if wui.canCreate(ctx, user) {
		if content != nil && !content.IsBinary() {
			rb.bindString("copy-url", sx.String(newURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionCopy).String()))
		}
		rb.bindString("version-url", sx.String(newURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionVersion).String()))
		rb.bindString("child-url", sx.String(newURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionChild).String()))
		rb.bindString("folge-url", sx.String(newURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionFolge).String()))
	}
	if wui.canRename(ctx, user, m) {
		rb.bindString("rename-url", sx.String(newURLBuilder('b').SetZid(apiZid).String()))
	}
	if wui.canDelete(ctx, user, m) {
		rb.bindString("delete-url", sx.String(newURLBuilder('d').SetZid(apiZid).String()))
	}
	if val, found := m.Get(api.KeyUselessFiles); found {
		rb.bindString("useless", sx.Cons(sx.String(val), nil))
	}
	rb.bindString("context-url", sx.String(newURLBuilder('h').AppendQuery(strZid+" "+api.ContextDirective).String()))
	if wui.canRefresh(user) {
		rb.bindString("reindex-url", sx.String(newURLBuilder('h').AppendQuery(strZid+" "+api.IdentDirective+api.ActionSeparator+"REINDEX").String()))
	}

	// Ensure to have title, role, tags, and syntax included as "meta-*"
	rb.bindKeyValue(api.KeyTitle, m.GetDefault(api.KeyTitle, ""))
	rb.bindKeyValue(api.KeyRole, m.GetDefault(api.KeyRole, ""))
	rb.bindKeyValue(api.KeyTags, m.GetDefault(api.KeyTags, ""))
	rb.bindKeyValue(api.KeySyntax, m.GetDefault(api.KeySyntax, ""))
	sentinel := sx.Cons(nil, nil)
	curr := sentinel
	for _, p := range m.ComputedPairs() {
		key, value := p.Key, p.Value
		curr = curr.AppendBang(sx.Cons(sx.String(key), sx.String(value)))

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
		text := sx.String(parser.NormalizedSpacedText(z.Meta.GetTitle()))
		link := sx.String(wui.NewURLBuilder('c').SetZid(api.ZettelID(zid.String())).
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

	wui.setSxnCache(zid, wui.engine.Rework(env, t))
	return t, nil
}
func (wui *WebUI) makeZettelReader(ctx context.Context, zid id.Zid) (*sxreader.Reader, error) {
	ztl, err := wui.box.GetZettel(ctx, zid)
	if err != nil {
		return nil, err
	}

	reader := sxreader.MakeReader(bytes.NewReader(ztl.Content.AsBytes()), sxreader.WithSymbolFactory(wui.sf))
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
		wui.log.Debug().Err(err).Msg("reportError")
	}
	user := server.GetUser(ctx)
	env, rb := wui.createRenderEnv(ctx, "error", api.ValueLangEN, "Error", user)
	rb.bindString("heading", sx.String(http.StatusText(code)))
	rb.bindString("message", sx.String(text))
	if rb.err == nil {
		rb.err = wui.renderSxnTemplateStatus(ctx, w, code, id.ErrorTemplateZid, env)
	}
	errSx := rb.err
	if errSx == nil {
		return
	}
	wui.log.Error().Err(errSx).Msg("while rendering error message")

	// if errBind != nil, the HTTP header was not written
	wui.prepareAndWriteHeader(w, http.StatusInternalServerError)
	fmt.Fprintf(
		w,
		`<!DOCTYPE html>
<html>
<head><title>Internal server error</title></head>
<body>
<h1>Internal server error</h1>
<p>When generating error code %d with message:</p><pre>%v</pre><p>an error occured:</p><pre>%v</pre>
</body>
</html>`, code, text, errSx)
}

func makeStringList(sl []string) *sx.Pair {
	if len(sl) == 0 {
		return nil
	}
	result := sx.Nil()
	for i := len(sl) - 1; i >= 0; i-- {
		result = result.Cons(sx.String(sl[i]))
	}
	return result
}
