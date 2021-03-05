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
	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/place"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/session"
)

// MakeGetHTMLZettelHandler creates a new HTTP handler for the use case "get zettel".
func MakeGetHTMLZettelHandler(
	te *TemplateEngine,
	parseZettel usecase.ParseZettel,
	getMeta usecase.GetMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			te.reportError(ctx, w, place.ErrNotFound)
			return
		}

		syntax := r.URL.Query().Get("syntax")
		zn, err := parseZettel.Run(ctx, zid, syntax)
		if err != nil {
			te.reportError(ctx, w, err)
			return
		}

		metaHeader, err := formatMeta(
			zn.InhMeta,
			"html",
			&encoder.StringsOption{
				Key:   "no-meta",
				Value: []string{meta.KeyTitle, meta.KeyLang},
			},
		)
		if err != nil {
			te.reportError(ctx, w, err)
			return
		}
		langOption := encoder.StringOption{Key: "lang", Value: runtime.GetLang(zn.InhMeta)}
		htmlTitle, err := adapter.FormatInlines(zn.Title, "html", &langOption)
		if err != nil {
			te.reportError(ctx, w, err)
			return
		}
		textTitle, err := adapter.FormatInlines(zn.Title, "text", &langOption)
		if err != nil {
			te.reportError(ctx, w, err)
			return
		}
		newWindow := true
		htmlContent, err := formatBlocks(
			zn.Ast,
			"html",
			&langOption,
			&encoder.StringOption{
				Key:   meta.KeyMarkerExternal,
				Value: runtime.GetMarkerExternal()},
			&encoder.BoolOption{Key: "newwindow", Value: newWindow},
			&encoder.AdaptLinkOption{
				Adapter: adapter.MakeLinkAdapter(ctx, 'h', getMeta, "", ""),
			},
			&encoder.AdaptImageOption{Adapter: adapter.MakeImageAdapter()},
		)
		if err != nil {
			te.reportError(ctx, w, err)
			return
		}
		user := session.GetUser(ctx)
		roleText := zn.Zettel.Meta.GetDefault(meta.KeyRole, "*")
		tags := buildTagInfos(zn.Zettel.Meta)
		getTitle := makeGetTitle(ctx, getMeta, &langOption)
		extURL, hasExtURL := zn.Zettel.Meta.Get(meta.KeyURL)
		backLinks := formatBackLinks(zn.InhMeta, getTitle)
		var base baseData
		te.makeBaseData(ctx, langOption.Value, textTitle, user, &base)
		base.MetaHeader = metaHeader
		canCopy := base.CanCreate && !zn.Zettel.Content.IsBinary()
		te.renderTemplate(ctx, w, id.ZettelTemplateZid, &base, struct {
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
			CanWrite:      te.canWrite(ctx, user, zn.Zettel),
			EditURL:       adapter.NewURLBuilder('e').SetZid(zid).String(),
			Zid:           zid.String(),
			InfoURL:       adapter.NewURLBuilder('i').SetZid(zid).String(),
			RoleText:      roleText,
			RoleURL:       adapter.NewURLBuilder('h').AppendQuery("role", roleText).String(),
			HasTags:       len(tags) > 0,
			Tags:          tags,
			CanCopy:       canCopy,
			CopyURL:       adapter.NewURLBuilder('c').SetZid(zid).String(),
			CanFolge:      base.CanCreate && !zn.Zettel.Content.IsBinary(),
			FolgeURL:      adapter.NewURLBuilder('f').SetZid(zid).String(),
			FolgeRefs:     formatMetaKey(zn.InhMeta, meta.KeyFolge, getTitle),
			PrecursorRefs: formatMetaKey(zn.InhMeta, meta.KeyPrecursor, getTitle),
			ExtURL:        extURL,
			HasExtURL:     hasExtURL,
			ExtNewWindow:  htmlAttrNewWindow(newWindow && hasExtURL),
			Content:       htmlContent,
			HasBackLinks:  len(backLinks) > 0,
			BackLinks:     backLinks,
		})
	}
}

func formatBlocks(
	bs ast.BlockSlice, format string, options ...encoder.Option) (string, error) {
	enc := encoder.Create(format, options...)
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

func formatMeta(m *meta.Meta, format string, options ...encoder.Option) (string, error) {
	enc := encoder.Create(format, options...)
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

func buildTagInfos(m *meta.Meta) []simpleLink {
	var tagInfos []simpleLink
	if tags, ok := m.GetList(meta.KeyTags); ok {
		ub := adapter.NewURLBuilder('h')
		tagInfos = make([]simpleLink, len(tags))
		for i, tag := range tags {
			tagInfos[i] = simpleLink{Text: tag, URL: ub.AppendQuery("tags", meta.CleanTag(tag)).String()}
			ub.ClearQuery()
		}
	}
	return tagInfos
}

func formatMetaKey(m *meta.Meta, key string, getTitle getTitleFunc) string {
	if _, ok := m.Get(key); ok {
		var buf bytes.Buffer
		writeHTMLMetaValue(&buf, m, key, getTitle, nil)
		return buf.String()
	}
	return ""
}

func formatBackLinks(m *meta.Meta, getTitle getTitleFunc) []simpleLink {
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
		if title, found := getTitle(zid, "text"); found > 0 {
			url := adapter.NewURLBuilder('h').SetZid(zid).String()
			if title == "" {
				result = append(result, simpleLink{Text: val, URL: url})
			} else {
				result = append(result, simpleLink{Text: title, URL: url})
			}
		}
	}
	return result
}
