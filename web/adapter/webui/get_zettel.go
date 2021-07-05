//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
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
	"net/http"
	"strings"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/encoder/encfun"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetHTMLZettelHandler creates a new HTTP handler for the use case "get zettel".
func (wui *WebUI) MakeGetHTMLZettelHandler(parseZettel usecase.ParseZettel, getMeta usecase.GetMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			wui.reportError(ctx, w, box.ErrNotFound)
			return
		}

		syntax := r.URL.Query().Get("syntax")
		zn, err := parseZettel.Run(ctx, zid, syntax)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}

		lang := config.GetLang(zn.InhMeta, wui.rtConfig)
		envHTML := encoder.Environment{
			LinkAdapter:    adapter.MakeLinkAdapter(ctx, wui, 'h', getMeta, "", encoder.EncoderUnknown),
			ImageAdapter:   adapter.MakeImageAdapter(ctx, wui, getMeta),
			CiteAdapter:    nil,
			Lang:           lang,
			Xhtml:          false,
			MarkerExternal: wui.rtConfig.GetMarkerExternal(),
			NewWindow:      true,
			IgnoreMeta:     map[string]bool{meta.KeyTitle: true, meta.KeyLang: true},
		}
		metaHeader, err := formatMeta(zn.InhMeta, encoder.EncoderHTML, &envHTML)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		htmlTitle, err := adapter.FormatInlines(
			encfun.MetaAsInlineSlice(zn.InhMeta, meta.KeyTitle), encoder.EncoderHTML, &envHTML)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		htmlContent, err := formatBlocks(zn.Ast, encoder.EncoderHTML, &envHTML)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		textTitle := encfun.MetaAsText(zn.InhMeta, meta.KeyTitle)
		user := wui.getUser(ctx)
		roleText := zn.Meta.GetDefault(meta.KeyRole, "*")
		tags := wui.buildTagInfos(zn.Meta)
		getTitle := makeGetTitle(ctx, getMeta, &encoder.Environment{Lang: lang})
		extURL, hasExtURL := zn.Meta.Get(meta.KeyURL)
		backLinks := wui.formatBackLinks(zn.InhMeta, getTitle)
		var base baseData
		wui.makeBaseData(ctx, lang, textTitle, user, &base)
		base.MetaHeader = metaHeader
		wui.renderTemplate(ctx, w, id.ZettelTemplateZid, &base, struct {
			HTMLTitle     string
			CanWrite      bool
			EditURL       string
			Zid           string
			InfoURL       string
			RoleText      string
			RoleURL       string
			HasTags       bool
			Tags          []simpleLink
			CanCopy       bool
			CopyURL       string
			CanFolge      bool
			FolgeURL      string
			FolgeRefs     string
			PrecursorRefs string
			HasExtURL     bool
			ExtURL        string
			ExtNewWindow  string
			Content       string
			HasBackLinks  bool
			BackLinks     []simpleLink
		}{
			HTMLTitle:     htmlTitle,
			CanWrite:      wui.canWrite(ctx, user, zn.Meta, zn.Content),
			EditURL:       wui.NewURLBuilder('e').SetZid(zid).String(),
			Zid:           zid.String(),
			InfoURL:       wui.NewURLBuilder('i').SetZid(zid).String(),
			RoleText:      roleText,
			RoleURL:       wui.NewURLBuilder('h').AppendQuery("role", roleText).String(),
			HasTags:       len(tags) > 0,
			Tags:          tags,
			CanCopy:       base.CanCreate && !zn.Content.IsBinary(),
			CopyURL:       wui.NewURLBuilder('c').SetZid(zid).String(),
			CanFolge:      base.CanCreate,
			FolgeURL:      wui.NewURLBuilder('f').SetZid(zid).String(),
			FolgeRefs:     wui.formatMetaKey(zn.InhMeta, meta.KeyFolge, getTitle),
			PrecursorRefs: wui.formatMetaKey(zn.InhMeta, meta.KeyPrecursor, getTitle),
			ExtURL:        extURL,
			HasExtURL:     hasExtURL,
			ExtNewWindow:  htmlAttrNewWindow(envHTML.NewWindow && hasExtURL),
			Content:       htmlContent,
			HasBackLinks:  len(backLinks) > 0,
			BackLinks:     backLinks,
		})
	}
}

func formatBlocks(bs ast.BlockSlice, format encoder.Enum, env *encoder.Environment) (string, error) {
	enc := encoder.Create(format, env)
	if enc == nil {
		return "", adapter.ErrNoSuchFormat
	}

	var content strings.Builder
	_, err := enc.WriteBlocks(&content, bs)
	if err != nil {
		return "", err
	}
	return content.String(), nil
}

func formatMeta(m *meta.Meta, format encoder.Enum, env *encoder.Environment) (string, error) {
	enc := encoder.Create(format, env)
	if enc == nil {
		return "", adapter.ErrNoSuchFormat
	}

	var content strings.Builder
	_, err := enc.WriteMeta(&content, m)
	if err != nil {
		return "", err
	}
	return content.String(), nil
}

func (wui *WebUI) buildTagInfos(m *meta.Meta) []simpleLink {
	var tagInfos []simpleLink
	if tags, ok := m.GetList(meta.KeyTags); ok {
		ub := wui.NewURLBuilder('h')
		tagInfos = make([]simpleLink, len(tags))
		for i, tag := range tags {
			tagInfos[i] = simpleLink{Text: tag, URL: ub.AppendQuery("tags", tag).String()}
			ub.ClearQuery()
		}
	}
	return tagInfos
}

func (wui *WebUI) formatMetaKey(m *meta.Meta, key string, getTitle getTitleFunc) string {
	if _, ok := m.Get(key); ok {
		var buf bytes.Buffer
		wui.writeHTMLMetaValue(&buf, m, key, getTitle, nil)
		return buf.String()
	}
	return ""
}

func (wui *WebUI) formatBackLinks(m *meta.Meta, getTitle getTitleFunc) []simpleLink {
	values, ok := m.GetList(meta.KeyBack)
	if !ok || len(values) == 0 {
		return nil
	}
	result := make([]simpleLink, 0, len(values))
	for _, val := range values {
		zid, err := id.Parse(val)
		if err != nil {
			continue
		}
		if title, found := getTitle(zid, encoder.EncoderText); found > 0 {
			url := wui.NewURLBuilder('h').SetZid(zid).String()
			if title == "" {
				result = append(result, simpleLink{Text: val, URL: url})
			} else {
				result = append(result, simpleLink{Text: title, URL: url})
			}
		}
	}
	return result
}
