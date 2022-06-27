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
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetCreateZettelHandler creates a new HTTP handler to display the
// HTML edit view for the various zettel creation methods.
func (wui *WebUI) MakeGetCreateZettelHandler(
	getZettel usecase.GetZettel, createZettel *usecase.CreateZettel,
	ucListRoles usecase.ListRoles, ucListSyntax usecase.ListSyntax) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		q := r.URL.Query()
		op := getCreateAction(q.Get(queryKeyAction))
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			wui.reportError(ctx, w, box.ErrNotFound)
			return
		}
		origZettel, err := getZettel.Run(box.NoEnrichContext(ctx), zid)
		if err != nil {
			wui.reportError(ctx, w, box.ErrNotFound)
			return
		}

		roleData, syntaxData := retrieveDataLists(ctx, ucListRoles, ucListSyntax)
		switch op {
		case actionCopy:
			wui.renderZettelForm(ctx, w, createZettel.PrepareCopy(origZettel), "Copy Zettel", "Copy Zettel", roleData, syntaxData)
		case actionFolge:
			wui.renderZettelForm(ctx, w, createZettel.PrepareFolge(origZettel), "Folge Zettel", "Folgezettel", roleData, syntaxData)
		case actionNew:
			m := origZettel.Meta
			title := parser.ParseMetadata(config.GetTitle(m, wui.rtConfig))
			textTitle, err2 := encodeInlinesText(&title, wui.gentext)
			if err2 != nil {
				wui.reportError(ctx, w, err2)
				return
			}
			htmlTitle, err2 := wui.getSimpleHTMLEncoder().InlinesString(&title, false)
			if err2 != nil {
				wui.reportError(ctx, w, err2)
				return
			}
			wui.renderZettelForm(ctx, w, createZettel.PrepareNew(origZettel), textTitle, htmlTitle, roleData, syntaxData)
		}
	}
}

func retrieveDataLists(ctx context.Context, ucListRoles usecase.ListRoles, ucListSyntax usecase.ListSyntax) ([]string, []string) {
	roleData := dataListFromArrangement(ucListRoles.Run(ctx))
	syntaxData := dataListFromArrangement(ucListSyntax.Run(ctx))
	return roleData, syntaxData
}

func dataListFromArrangement(ar meta.Arrangement, err error) []string {
	l := ar.Counted()
	l.SortByCount()
	return l.Categories()
}

func (wui *WebUI) renderZettelForm(
	ctx context.Context,
	w http.ResponseWriter,
	zettel domain.Zettel,
	title, heading string,
	roleData []string,
	syntaxData []string,
) {
	user := wui.getUser(ctx)
	m := zettel.Meta
	var base baseData
	wui.makeBaseData(ctx, config.GetLang(m, wui.rtConfig), title, "", user, &base)
	wui.renderTemplate(ctx, w, id.FormTemplateZid, &base, formZettelData{
		Heading:       heading,
		MetaTitle:     m.GetDefault(api.KeyTitle, ""),
		MetaTags:      m.GetDefault(api.KeyTags, ""),
		MetaRole:      config.GetRole(m, wui.rtConfig),
		HasRoleData:   len(roleData) > 0,
		RoleData:      roleData,
		HasSyntaxData: len(syntaxData) > 0,
		SyntaxData:    syntaxData,
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
		reEdit, zettel, hasContent, err := parseZettelForm(r, id.Invalid)
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
		if reEdit {
			wui.redirectFound(w, r, wui.NewURLBuilder('e').SetZid(api.ZettelID(newZid.String())))
		} else {
			wui.redirectFound(w, r, wui.NewURLBuilder('h').SetZid(api.ZettelID(newZid.String())))
		}
	}
}
