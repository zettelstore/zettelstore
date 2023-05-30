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
	"context"
	"net/http"

	"codeberg.org/t73fde/sxpf"
	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/server"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

// MakeGetHTMLZettelHandler creates a new HTTP handler for the use case "get zettel".
func (wui *WebUI) MakeGetHTMLZettelHandler(evaluate *usecase.Evaluate, getMeta usecase.GetMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			wui.reportError(ctx, w, box.ErrNotFound)
			return
		}

		q := r.URL.Query()
		zn, err := evaluate.Run(ctx, zid, q.Get(api.KeySyntax))
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}

		enc := wui.getSimpleHTMLEncoder()
		metaObj := enc.MetaSxn(zn.InhMeta, createEvalMetadataFunc(ctx, evaluate))
		content, endnotes, err := enc.BlocksSxn(&zn.Ast)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}

		cssRoleURL, err := wui.getCSSRoleURL(ctx, zn.InhMeta)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}

		user := server.GetUser(ctx)
		apiZid := api.ZettelID(zid.String())
		getTextTitle := wui.makeGetTextTitle(ctx, getMeta)

		title := parser.NormalizedSpacedText(zn.InhMeta.GetTitle())
		env, err := wui.createRenderEnv(ctx, "zettel", wui.rtConfig.Get(ctx, zn.InhMeta, api.KeyLang), title, user)
		rb := makeRenderBinder(wui.sf, env, err)
		rb.bindSymbol(wui.symMetaHeader, metaObj)
		rb.bindString("css-role-url", sxpf.MakeString(cssRoleURL))
		rb.bindString("heading", sxpf.MakeString(title))
		if wui.canWrite(ctx, user, zn.Meta, zn.Content) {
			rb.bindString("edit-url", sxpf.MakeString(wui.NewURLBuilder('e').SetZid(apiZid).String()))
		}
		rb.bindString("info-url", sxpf.MakeString(wui.NewURLBuilder('i').SetZid(apiZid).String()))
		rb.bindString("role-url",
			sxpf.MakeString(wui.NewURLBuilder('h').AppendQuery(api.KeyRole+api.SearchOperatorHas+zn.Meta.GetDefault(api.KeyRole, "")).String()))
		rb.bindString("tag-refs", wui.transformTagSet(api.KeyTags, meta.ListFromValue(zn.InhMeta.GetDefault(api.KeyTags, ""))))
		if wui.canCreate(ctx, user) {
			if !zn.Content.IsBinary() {
				rb.bindString("copy-url", sxpf.MakeString(wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionCopy).String()))
			}
			rb.bindString("version-url", sxpf.MakeString(wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionVersion).String()))
			rb.bindString("folge-url", sxpf.MakeString(wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionFolge).String()))
		}
		rb.bindString("predecessor-refs", wui.identifierSetAsLinks(zn.InhMeta, api.KeyPredecessor, getTextTitle))
		rb.bindString("precursor-refs", wui.identifierSetAsLinks(zn.InhMeta, api.KeyPrecursor, getTextTitle))
		rb.bindString("superior-refs", wui.identifierSetAsLinks(zn.InhMeta, api.KeySuperior, getTextTitle))
		rb.bindString("ext-url", wui.urlFromMeta(zn.InhMeta, api.KeyURL))
		rb.bindString("content", content)
		rb.bindString("endnotes", endnotes)
		rb.bindString("folge-links", wui.zettelLinksSxn(zn.InhMeta, api.KeyFolge, getTextTitle))
		rb.bindString("subordinate-links", wui.zettelLinksSxn(zn.InhMeta, api.KeySubordinates, getTextTitle))
		rb.bindString("back-links", wui.zettelLinksSxn(zn.InhMeta, api.KeyBack, getTextTitle))
		rb.bindString("successor-links", wui.zettelLinksSxn(zn.InhMeta, api.KeySuccessors, getTextTitle))
		if rb.err == nil {
			err = bindMeta(zn.InhMeta, wui.sf, env)
		}
		if err == nil {
			err = wui.renderSxnTemplate(ctx, w, id.ZettelTemplateZid, env)
		}
		if err != nil {
			wui.reportError(ctx, w, err)
		}
	}
}

func (wui *WebUI) getCSSRoleURL(ctx context.Context, m *meta.Meta) (string, error) {
	cssZid, err := wui.retrieveCSSZidFromRole(ctx, m)
	if err != nil {
		return "", err
	}
	if cssZid == id.Invalid {
		return "", nil
	}
	return wui.NewURLBuilder('z').SetZid(api.ZettelID(cssZid.String())).String(), nil
}

func (wui *WebUI) identifierSetAsLinks(m *meta.Meta, key string, getTextTitle getTextTitleFunc) *sxpf.List {
	if values, ok := m.GetList(key); ok {
		return wui.transformIdentifierSet(values, getTextTitle)
	}
	return sxpf.Nil()
}

func (wui *WebUI) urlFromMeta(m *meta.Meta, key string) sxpf.Object {
	val, found := m.Get(key)
	if !found || val == "" {
		return sxpf.Nil()
	}
	return wui.transformURL(val)
}

func (wui *WebUI) zettelLinksSxn(m *meta.Meta, key string, getTextTitle getTextTitleFunc) *sxpf.List {
	values, ok := m.GetList(key)
	if !ok || len(values) == 0 {
		return nil
	}
	return wui.zidLinksSxn(values, getTextTitle)
}

func (wui *WebUI) zidLinksSxn(values []string, getTextTitle getTextTitleFunc) (lst *sxpf.List) {
	for i := len(values) - 1; i >= 0; i-- {
		val := values[i]
		zid, err := id.Parse(val)
		if err != nil {
			continue
		}
		if title, found := getTextTitle(zid); found > 0 {
			url := sxpf.MakeString(wui.NewURLBuilder('h').SetZid(api.ZettelID(zid.String())).String())
			if title == "" {
				lst = lst.Cons(sxpf.Cons(sxpf.MakeString(val), url))
			} else {
				lst = lst.Cons(sxpf.Cons(sxpf.MakeString(title), url))
			}
		}
	}
	return lst
}

func bindMeta(m *meta.Meta, sf sxpf.SymbolFactory, env sxpf.Environment) error {
	for _, p := range m.ComputedPairs() {
		keySym, err := sf.Make("meta-" + p.Key)
		if err != nil {
			return err
		}
		if kt := meta.Type(p.Key); kt.IsSet {
			values := meta.ListFromValue(p.Value)
			if len(values) == 0 {
				continue
			}
			sxValues := make([]sxpf.Object, len(values))
			for i, v := range values {
				sxValues[i] = sxpf.MakeString(v)
			}
			err = env.Bind(keySym, sxpf.MakeList(sxValues...))
		} else {
			err = env.Bind(keySym, sxpf.MakeString(p.Value))
		}
		if err != nil {
			return err
		}
	}
	symZid, err := sf.Make("zid")
	if err != nil {
		return err
	}
	return env.Bind(symZid, sxpf.MakeString(m.Zid.String()))
}
