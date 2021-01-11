//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package webui provides wet-UI handlers for web requests.
package webui

import (
	"net/http"
	"strings"

	"zettelstore.de/z/ast"
	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
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
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		syntax := r.URL.Query().Get("syntax")
		zn, err := parseZettel.Run(ctx, zid, syntax)
		if err != nil {
			adapter.ReportUsecaseError(w, err)
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
			adapter.InternalServerError(w, "Format meta", err)
			return
		}
		langOption := encoder.StringOption{Key: "lang", Value: runtime.GetLang(zn.InhMeta)}
		htmlTitle, err := adapter.FormatInlines(zn.Title, "html", &langOption)
		if err != nil {
			adapter.InternalServerError(w, "Format HTML inlines", err)
			return
		}
		textTitle, err := adapter.FormatInlines(zn.Title, "text", &langOption)
		if err != nil {
			adapter.InternalServerError(w, "Format text inlines", err)
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
			adapter.InternalServerError(w, "Format blocks", err)
			return
		}
		user := session.GetUser(ctx)
		roleText := zn.Zettel.Meta.GetDefault(meta.KeyRole, "*")
		tags := buildTagInfos(zn.Zettel.Meta)
		extURL, hasExtURL := zn.Zettel.Meta.Get(meta.KeyURL)
		var base baseData
		te.makeBaseData(ctx, langOption.Value, textTitle, user, &base)
		base.MetaHeader = metaHeader
		canCopy := base.CanCreate && !zn.Zettel.Content.IsBinary()
		te.renderTemplate(ctx, w, id.DetailTemplateZid, &base, struct {
			HTMLTitle    string
			CanWrite     bool
			EditURL      string
			Zid          string
			InfoURL      string
			RoleText     string
			RoleURL      string
			HasTags      bool
			Tags         []simpleLink
			CanCopy      bool
			CopyURL      string
			CanNew       bool
			NewURL       string
			CanFolge     bool
			FolgeURL     string
			HasExtURL    bool
			ExtURL       string
			ExtNewWindow string
			Content      string
		}{
			HTMLTitle:    htmlTitle,
			CanWrite:     te.canWrite(ctx, user, zn.Zettel),
			EditURL:      adapter.NewURLBuilder('e').SetZid(zid).String(),
			Zid:          zid.String(),
			InfoURL:      adapter.NewURLBuilder('i').SetZid(zid).String(),
			RoleText:     roleText,
			RoleURL:      adapter.NewURLBuilder('h').AppendQuery("role", roleText).String(),
			HasTags:      len(tags) > 0,
			Tags:         tags,
			CanCopy:      canCopy,
			CopyURL:      adapter.NewURLBuilder('c').SetZid(zid).String(),
			CanNew:       canCopy && roleText == meta.ValueRoleNewTemplate,
			NewURL:       adapter.NewURLBuilder('n').SetZid(zid).String(),
			CanFolge:     base.CanCreate && !zn.Zettel.Content.IsBinary(),
			FolgeURL:     adapter.NewURLBuilder('f').SetZid(zid).String(),
			ExtURL:       extURL,
			HasExtURL:    hasExtURL,
			ExtNewWindow: htmlAttrNewWindow(newWindow && hasExtURL),
			Content:      htmlContent,
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
		tagInfos = make([]simpleLink, 0, len(tags))
		ub := adapter.NewURLBuilder('h')
		for _, t := range tags {
			// Cast to template.HTML is ok, because "t" is a tag name
			// and contains only legal characters by construction.
			tagInfos = append(
				tagInfos,
				simpleLink{Text: t, URL: ub.AppendQuery("tags", t).String()})
			ub.ClearQuery()
		}
	}
	return tagInfos
}
