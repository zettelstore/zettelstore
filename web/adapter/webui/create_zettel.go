//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
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
	"fmt"
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetCopyZettelHandler creates a new HTTP handler to display the
// HTML edit view of a copied zettel.
func (wui *WebUI) MakeGetCopyZettelHandler(getZettel usecase.GetZettel, copyZettel usecase.CopyZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		origZettel, err := getOrigZettel(ctx, r, getZettel, "Copy")
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
		origZettel, err := getOrigZettel(ctx, r, getZettel, "Folge")
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
		origZettel, err := getOrigZettel(ctx, r, getZettel, "New")
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		m := origZettel.Meta
		title := parser.ParseMetadata(config.GetTitle(m, wui.rtConfig))
		textTitle, err := encodeInlines(&title, api.EncoderText, nil)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		env := encoder.Environment{Lang: config.GetLang(m, wui.rtConfig)}
		htmlTitle, err := encodeInlines(&title, api.EncoderHTML, &env)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		wui.renderZettelForm(w, r, newZettel.Run(origZettel), textTitle, htmlTitle)
	}
}

func getOrigZettel(
	ctx context.Context,
	r *http.Request,
	getZettel usecase.GetZettel,
	op string,
) (domain.Zettel, error) {
	if enc, encText := adapter.GetEncoding(r, r.URL.Query(), api.EncoderHTML); enc != api.EncoderHTML {
		return domain.Zettel{}, adapter.NewErrBadRequest(
			fmt.Sprintf("%v zettel not possible in encoding %q", op, encText))
	}
	zid, err := id.Parse(r.URL.Path[1:])
	if err != nil {
		return domain.Zettel{}, box.ErrNotFound
	}
	origZettel, err := getZettel.Run(box.NoEnrichContext(ctx), zid)
	if err != nil {
		return domain.Zettel{}, box.ErrNotFound
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
	user := wui.getUser(ctx)
	m := zettel.Meta
	var base baseData
	wui.makeBaseData(ctx, config.GetLang(m, wui.rtConfig), title, user, &base)
	wui.renderTemplate(ctx, w, id.FormTemplateZid, &base, formZettelData{
		Heading:       heading,
		MetaTitle:     m.GetDefault(api.KeyTitle, ""),
		MetaTags:      m.GetDefault(api.KeyTags, ""),
		MetaRole:      config.GetRole(m, wui.rtConfig),
		MetaSyntax:    config.GetSyntax(m, wui.rtConfig),
		MetaPairsRest: m.PairsRest(),
		IsTextContent: !zettel.Content.IsBinary(),
		Content:       zettel.Content.AsString(),
	})
}

// MakePostCreateZettelHandler creates a new HTTP handler to store content of
// an existing zettel.
func (wui *WebUI) MakePostCreateZettelHandler(createZettel *usecase.CreateZettel) http.HandlerFunc {
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
		wui.redirectFound(w, r, wui.NewURLBuilder('h').SetZid(api.ZettelID(newZid.String())))
	}
}
