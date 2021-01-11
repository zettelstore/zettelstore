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
	"fmt"
	"net/http"

	"zettelstore.de/z/config/runtime"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"

	"zettelstore.de/z/domain"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
	"zettelstore.de/z/web/session"
)

// MakeGetCopyZettelHandler creates a new HTTP handler to display the
// HTML edit view of a copied zettel.
func MakeGetCopyZettelHandler(
	te *TemplateEngine,
	getZettel usecase.GetZettel,
	copyZettel usecase.CopyZettel,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if origZettel, ok := getOrigZettel(w, r, getZettel, "Copy"); ok {
			renderZettelForm(
				w,
				r,
				te,
				copyZettel.Run(origZettel), "Copy Zettel", "Copy Zettel")
		}
	}
}

// MakeGetFolgeZettelHandler creates a new HTTP handler to display the
// HTML edit view of a follow-up zettel.
func MakeGetFolgeZettelHandler(
	te *TemplateEngine,
	getZettel usecase.GetZettel,
	folgeZettel usecase.FolgeZettel,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if origZettel, ok := getOrigZettel(w, r, getZettel, "Folge"); ok {
			renderZettelForm(
				w,
				r,
				te,
				folgeZettel.Run(origZettel), "Folge Zettel", "Folgezettel")
		}
	}
}

// MakeGetNewZettelHandler creates a new HTTP handler to display the
// HTML edit view of a zettel.
func MakeGetNewZettelHandler(
	te *TemplateEngine,
	getZettel usecase.GetZettel,
	newZettel usecase.NewZettel,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if origZettel, ok := getOrigZettel(w, r, getZettel, "New"); ok {
			m := origZettel.Meta
			title := parser.ParseInlines(
				input.NewInput(runtime.GetTitle(m)), meta.ValueSyntaxZmk)
			langOption := encoder.StringOption{Key: "lang", Value: runtime.GetLang(m)}
			textTitle, err := adapter.FormatInlines(title, "text", &langOption)
			if err != nil {
				adapter.InternalServerError(w, "Format Text inlines for WebUI", err)
				return
			}
			htmlTitle, err := adapter.FormatInlines(title, "html", &langOption)
			if err != nil {
				adapter.InternalServerError(w, "Format HTML inlines for WebUI", err)
				return
			}
			renderZettelForm(
				w, r, te, newZettel.Run(origZettel), textTitle, htmlTitle)
		}
	}
}

func getOrigZettel(
	w http.ResponseWriter,
	r *http.Request,
	getZettel usecase.GetZettel,
	op string,
) (domain.Zettel, bool) {
	if format := adapter.GetFormat(r, r.URL.Query(), "html"); format != "html" {
		adapter.BadRequest(w, fmt.Sprintf("%v zettel not possible in format %q", op, format))
		return domain.Zettel{}, false
	}
	zid, err := id.Parse(r.URL.Path[1:])
	if err != nil {
		http.NotFound(w, r)
		return domain.Zettel{}, false
	}
	origZettel, err := getZettel.Run(r.Context(), zid)
	if err != nil {
		http.NotFound(w, r)
		return domain.Zettel{}, false
	}
	return origZettel, true
}

func renderZettelForm(
	w http.ResponseWriter,
	r *http.Request,
	te *TemplateEngine,
	zettel domain.Zettel,
	title string,
	heading string,
) {
	ctx := r.Context()
	user := session.GetUser(ctx)
	m := zettel.Meta
	var base baseData
	te.makeBaseData(ctx, runtime.GetLang(m), title, user, &base)
	te.renderTemplate(r.Context(), w, id.FormTemplateZid, &base, formZettelData{
		Heading:       heading,
		MetaTitle:     runtime.GetTitle(m),
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
func MakePostCreateZettelHandler(createZettel usecase.CreateZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zettel, hasContent, err := parseZettelForm(r, id.Invalid)
		if err != nil {
			adapter.BadRequest(w, "Unable to read form data")
			return
		}
		if !hasContent {
			adapter.BadRequest(w, "Content is missing")
			return
		}

		if newZid, err := createZettel.Run(r.Context(), zettel); err != nil {
			adapter.ReportUsecaseError(w, err)
		} else {
			http.Redirect(
				w, r, adapter.NewURLBuilder('h').SetZid(newZid).String(), http.StatusFound)
		}
	}
}
