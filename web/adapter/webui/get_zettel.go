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
	"strings"

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
		htmlContent, err := enc.BlocksString(&zn.Ast)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		var roleCSSURL string
		cssZid, err := wui.retrieveCSSZidFromRole(ctx, zn.InhMeta)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		if cssZid != id.Invalid {
			roleCSSURL = wui.NewURLBuilder('z').SetZid(api.ZettelID(cssZid.String())).String()
		}
		user := server.GetUser(ctx)
		roleText := zn.Meta.GetDefault(api.KeyRole, "")
		canCreate := wui.canCreate(ctx, user)
		getTextTitle := wui.makeGetTextTitle(ctx, getMeta)
		extURL, hasExtURL := wui.formatURLFromMeta(zn.Meta, api.KeyURL)
		folgeLinks := createSimpleLinks(wui.encodeZettelLinks(zn.InhMeta, api.KeyFolge, getTextTitle))
		subordinates := createSimpleLinks(wui.encodeZettelLinks(zn.InhMeta, api.KeySubordinates, getTextTitle))
		backLinks := createSimpleLinks(wui.encodeZettelLinks(zn.InhMeta, api.KeyBack, getTextTitle))
		successorLinks := createSimpleLinks(wui.encodeZettelLinks(zn.InhMeta, api.KeySuccessors, getTextTitle))
		apiZid := api.ZettelID(zid.String())

		var base baseData
		title := parser.NormalizedSpacedText(zn.InhMeta.GetTitle())
		wui.makeBaseData(ctx, wui.rtConfig.Get(ctx, zn.InhMeta, api.KeyLang), title, roleCSSURL, user, &base)
		base.MetaHeader = enc.MetaString(zn.InhMeta, createEvalMetadataFunc(ctx, evaluate))
		wui.renderTemplate(ctx, w, id.ZettelTemplateZid, &base, struct {
			Heading         string
			RoleCSS         string
			CanWrite        bool
			EditURL         string
			Zid             string
			InfoURL         string
			RoleText        string
			RoleURL         string
			Tags            simpleLinks
			CanCopy         bool
			CopyURL         string
			CanVersion      bool
			VersionURL      string
			CanFolge        bool
			FolgeURL        string
			PredecessorRefs string
			PrecursorRefs   string
			SuperiorRefs    string
			HasExtURL       bool
			ExtURL          string
			Author          string
			Content         string
			NeedBottomNav   bool
			FolgeLinks      simpleLinks
			Subordinates    simpleLinks
			BackLinks       simpleLinks
			SuccessorLinks  simpleLinks
		}{
			Heading:         title,
			RoleCSS:         roleCSSURL,
			CanWrite:        wui.canWrite(ctx, user, zn.Meta, zn.Content),
			EditURL:         wui.NewURLBuilder('e').SetZid(apiZid).String(),
			Zid:             zid.String(),
			InfoURL:         wui.NewURLBuilder('i').SetZid(apiZid).String(),
			RoleText:        roleText,
			RoleURL:         wui.NewURLBuilder('h').AppendQuery(api.KeyRole + api.SearchOperatorHas + roleText).String(),
			Tags:            createSimpleLinks(wui.buildTagInfos(zn.Meta)),
			CanCopy:         canCreate && !zn.Content.IsBinary(),
			CopyURL:         wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionCopy).String(),
			CanVersion:      canCreate,
			VersionURL:      wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionVersion).String(),
			CanFolge:        canCreate,
			FolgeURL:        wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionFolge).String(),
			PredecessorRefs: wui.encodeIdentifierSet(zn.InhMeta, api.KeyPredecessor, getTextTitle),
			PrecursorRefs:   wui.encodeIdentifierSet(zn.InhMeta, api.KeyPrecursor, getTextTitle),
			SuperiorRefs:    wui.encodeIdentifierSet(zn.InhMeta, api.KeySuperior, getTextTitle),
			ExtURL:          extURL,
			HasExtURL:       hasExtURL,
			Author:          zn.Meta.GetDefault(api.KeyAuthor, ""),
			Content:         htmlContent,
			NeedBottomNav:   folgeLinks.Has || subordinates.Has || backLinks.Has || successorLinks.Has,
			FolgeLinks:      folgeLinks,
			Subordinates:    subordinates,
			BackLinks:       backLinks,
			SuccessorLinks:  successorLinks,
		})
	}
}

func (wui *WebUI) formatURLFromMeta(m *meta.Meta, key string) (string, bool) {
	val, found := m.Get(key)
	if !found {
		return "", false
	}
	if val == "" {
		return "", false
	}

	var sb strings.Builder
	_, err := wui.htmlGen.WriteHTML(&sb, wui.transformURL(val))
	if err != nil {
		return "", false
	}
	return sb.String(), true
}

func (wui *WebUI) buildTagInfos(m *meta.Meta) []simpleLink {
	var tagInfos []simpleLink
	if tags, ok := m.GetList(api.KeyTags); ok {
		ub := wui.NewURLBuilder('h')
		tagInfos = make([]simpleLink, len(tags))
		for i, tag := range tags {
			tagInfos[i] = simpleLink{Text: tag, URL: ub.AppendQuery(api.KeyTags + api.SearchOperatorHas + tag).String()}
			ub.ClearQuery()
		}
	}
	return tagInfos
}

func (wui *WebUI) encodeIdentifierSet(m *meta.Meta, key string, getTextTitle getTextTitleFunc) string {
	if values, ok := m.GetList(key); ok {
		sval := wui.transformIdentifierSet(values, getTextTitle)
		var sb strings.Builder
		wui.htmlGen.WriteHTML(&sb, sval)
		return sb.String()
	}
	return ""
}

func (wui *WebUI) encodeZettelLinks(m *meta.Meta, key string, getTextTitle getTextTitleFunc) []simpleLink {
	values, ok := m.GetList(key)
	if !ok || len(values) == 0 {
		return nil
	}
	return wui.encodeZidLinks(values, getTextTitle)
}

func (wui *WebUI) encodeZidLinks(values []string, getTextTitle getTextTitleFunc) []simpleLink {
	result := make([]simpleLink, 0, len(values))
	for _, val := range values {
		zid, err := id.Parse(val)
		if err != nil {
			continue
		}
		if title, found := getTextTitle(zid); found > 0 {
			url := wui.NewURLBuilder('h').SetZid(api.ZettelID(zid.String())).String()
			if title == "" {
				result = append(result, simpleLink{Text: val, URL: url})
			} else {
				result = append(result, simpleLink{Text: title, URL: url})
			}
		}
	}
	return result
}

// --------------------------------------------------------------------------------------
// Below is experimental code that will render a zettel with the help of an SXN template.
//
// If successful, it will replace above code.
// --------------------------------------------------------------------------------------

// MakeGetHTMLZettelHandlerSxn creates a new HTTP handler for the use case "get zettel".
func (wui *WebUI) MakeGetHTMLZettelHandlerSxn(evaluate *usecase.Evaluate, getMeta usecase.GetMeta) http.HandlerFunc {
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
		canCreate := wui.canCreate(ctx, user)
		getTextTitle := wui.makeGetTextTitle(ctx, getMeta)

		lang := wui.rtConfig.Get(ctx, zn.InhMeta, api.KeyLang)
		title := parser.NormalizedSpacedText(zn.InhMeta.GetTitle())
		env, err := wui.createRenderEnv(ctx, wui.engine.RootEnvironment(), "zettel", lang, title, user)
		if err == nil {
			err = env.Bind(wui.symMetaHeader, metaObj)
		}
		sf := wui.sf
		err = bindRenderEnv(err, env, sf, "css-role-url", sxpf.MakeString(cssRoleURL))
		err = bindRenderEnv(err, env, sf, "heading", sxpf.MakeString(title))
		err = bindRenderEnv(err, env, sf, "can-write", sxpf.MakeBoolean(wui.canWrite(ctx, user, zn.Meta, zn.Content)))
		err = bindRenderEnv(err, env, sf, "edit-url", sxpf.MakeString(wui.NewURLBuilder('e').SetZid(apiZid).String()))
		err = bindRenderEnv(err, env, sf, "info-url", sxpf.MakeString(wui.NewURLBuilder('i').SetZid(apiZid).String()))
		err = bindRenderEnv(err, env, sf, "role-url",
			sxpf.MakeString(wui.NewURLBuilder('h').AppendQuery(api.KeyRole+api.SearchOperatorHas+zn.Meta.GetDefault(api.KeyRole, "")).String()))
		err = bindRenderEnv(err, env, sf, "tag-refs", wui.transformTagSet(api.KeyTags, meta.ListFromValue(zn.InhMeta.GetDefault(api.KeyTags, ""))))
		err = bindRenderEnv(err, env, sf, "can-copy", sxpf.Boolean(canCreate && !zn.Content.IsBinary()))
		err = bindRenderEnv(err, env, sf, "copy-url", sxpf.MakeString(wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionCopy).String()))
		err = bindRenderEnv(err, env, sf, "can-version", sxpf.Boolean(canCreate))
		err = bindRenderEnv(err, env, sf, "version-url", sxpf.MakeString(wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionVersion).String()))
		err = bindRenderEnv(err, env, sf, "can-folge", sxpf.Boolean(canCreate))
		err = bindRenderEnv(err, env, sf, "folge-url", sxpf.MakeString(wui.NewURLBuilder('c').SetZid(apiZid).AppendKVQuery(queryKeyAction, valueActionFolge).String()))
		err = bindRenderEnv(err, env, sf, "predecessor-refs", wui.identifierSetAsLinks(zn.InhMeta, api.KeyPredecessor, getTextTitle))
		err = bindRenderEnv(err, env, sf, "precursor-refs", wui.identifierSetAsLinks(zn.InhMeta, api.KeyPrecursor, getTextTitle))
		err = bindRenderEnv(err, env, sf, "superior-refs", wui.identifierSetAsLinks(zn.InhMeta, api.KeySuperior, getTextTitle))
		err = bindRenderEnv(err, env, sf, "ext-url", wui.urlFromMeta(zn.InhMeta, api.KeyURL))
		err = bindRenderEnv(err, env, sf, "content", content)
		err = bindRenderEnv(err, env, sf, "endnotes", endnotes)
		// folgeLinks := createSimpleLinks(wui.encodeZettelLinks(zn.InhMeta, api.KeyFolge, getTextTitle))
		// subordinates := createSimpleLinks(wui.encodeZettelLinks(zn.InhMeta, api.KeySubordinates, getTextTitle))
		// backLinks := createSimpleLinks(wui.encodeZettelLinks(zn.InhMeta, api.KeyBack, getTextTitle))
		// successorLinks := createSimpleLinks(wui.encodeZettelLinks(zn.InhMeta, api.KeySuccessors, getTextTitle))
		// 	NeedBottomNav:   folgeLinks.Has || subordinates.Has || backLinks.Has || successorLinks.Has,
		// 	FolgeLinks:      folgeLinks,
		// 	Subordinates:    subordinates,
		// 	BackLinks:       backLinks,
		// 	SuccessorLinks:  successorLinks,
		if err == nil {
			err = bindMeta(zn.InhMeta, sf, env)
		}
		if err != nil {
			wui.reportError(ctx, w, err) // TODO: template might throw error, write basic HTML page w/o template
			return
		}

		err = wui.renderSxnTemplate(ctx, w, id.ZettelTemplateZid+30000, env)
		if err != nil {
			wui.reportError(ctx, w, err) // TODO: template might throw error, write basic HTML page w/o template
			return
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
