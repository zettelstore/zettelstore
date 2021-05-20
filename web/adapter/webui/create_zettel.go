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
	"context"
	"fmt"
	"net/http"

	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/input"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/place"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetCopyZettelHandler creates a new HTTP handler to display the
// HTML edit view of a copied zettel.
func (wui *WebUI) MakeGetCopyZettelHandler(getZettel usecase.GetZettel, copyZettel usecase.CopyZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		origZettel, err := getOrigZettel(ctx, w, r, getZettel, "Copy")
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		wui.renderZettelForm(w, r, copyZettel.Run(origZettel), "Copy Zettel", "Copy Zettel")
	}
}

// MakeGetFolgeZettelHandler creates a new HTTP handler to display the
// HTML edit view of a follow-up zettel.
func (wui *WebUI) MakeGetFolgeZettelHandler(getZettel usecase.GetZettel, folgeZettel usecase.FolgeZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		origZettel, err := getOrigZettel(ctx, w, r, getZettel, "Folge")
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		wui.renderZettelForm(w, r, folgeZettel.Run(origZettel), "Folge Zettel", "Folgezettel")
	}
}

// MakeGetNewZettelHandler creates a new HTTP handler to display the
// HTML edit view of a zettel.
func (wui *WebUI) MakeGetNewZettelHandler(getZettel usecase.GetZettel, newZettel usecase.NewZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		origZettel, err := getOrigZettel(ctx, w, r, getZettel, "New")
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		m := origZettel.Meta
		title := parser.ParseInlines(input.NewInput(runtime.GetTitle(m)), meta.ValueSyntaxZmk)
		textTitle, err := adapter.FormatInlines(title, "text", nil)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		env := encoder.Environment{Lang: runtime.GetLang(m)}
		htmlTitle, err := adapter.FormatInlines(title, "html", &env)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		wui.renderZettelForm(w, r, newZettel.Run(origZettel), textTitle, htmlTitle)
	}
}

func getOrigZettel(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	getZettel usecase.GetZettel,
	op string,
) (domain.Zettel, error) {
	if format := adapter.GetFormat(r, r.URL.Query(), "html"); format != "html" {
		return domain.Zettel{}, adapter.NewErrBadRequest(
			fmt.Sprintf("%v zettel not possible in format %q", op, format))
	}
	zid, err := id.Parse(r.URL.Path[1:])
	if err != nil {
		return domain.Zettel{}, place.ErrNotFound
	}
	origZettel, err := getZettel.Run(place.NoEnrichContext(ctx), zid)
	if err != nil {
		return domain.Zettel{}, place.ErrNotFound
	}
	return origZettel, nil
}

func (wui *WebUI) renderZettelForm(
	w http.ResponseWriter,
	r *http.Request,
	zettel domain.Zettel,
	title, heading string,
) {
	ctx := r.Context()
	user := wui.ab.GetUser(ctx)
	m := zettel.Meta
	var base baseData
	wui.makeBaseData(ctx, runtime.GetLang(m), title, user, &base)
	wui.renderTemplate(ctx, w, id.FormTemplateZid, &base, formZettelData{
		Heading:       heading,
		MetaTitle:     m.GetDefault(meta.KeyTitle, ""),
		MetaTags:      m.GetDefault(meta.KeyTags, ""),
		MetaRole:      runtime.GetRole(m),
		MetaSyntax:    runtime.GetSyntax(m),
		MetaPairsRest: m.PairsRest(false),
		IsTextContent: !zettel.Content.IsBinary(),
		Content:       zettel.Content.AsString(),
	})
}

// MakePostCreateZettelHandler creates a new HTTP handler to store content of
// an existing zettel.
func (wui *WebUI) MakePostCreateZettelHandler(createZettel usecase.CreateZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		zettel, hasContent, err := parseZettelForm(r, id.Invalid)
		if err != nil {
			wui.reportError(ctx, w, adapter.NewErrBadRequest("Unable to read form data"))
			return
		}
		if !hasContent {
			wui.reportError(ctx, w, adapter.NewErrBadRequest("Content is missing"))
			return
		}

		newZid, err := createZettel.Run(ctx, zettel)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		redirectFound(w, r, wui.ab.NewURLBuilder('h').SetZid(newZid))
	}
}
