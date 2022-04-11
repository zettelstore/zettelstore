//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package webui

import (
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/z/ast"
	"zettelstore.de/z/domain/id"
)

func (wui *WebUI) redirectFound(w http.ResponseWriter, r *http.Request, ub *api.URLBuilder) {
	us := ub.String()
	wui.log.Debug().Str("uri", us).Msg("redirect")
	http.Redirect(w, r, us, http.StatusFound)
}

// createTagReference builds a reference to list all tags.
func (wui *WebUI) createTagReference(s string) *ast.Reference {
	u := wui.NewURLBuilder('h').AppendQuery(api.QueryKeyEncoding, api.EncodingHTML).AppendQuery(api.KeyAllTags, s)
	ref := ast.ParseReference(u.String())
	ref.State = ast.RefStateHosted
	return ref
}

// createHostedReference builds a reference with state "hosted".
func (wui *WebUI) createHostedReference(s string) *ast.Reference {
	urlPrefix := wui.GetURLPrefix()
	ref := ast.ParseReference(urlPrefix + s)
	ref.State = ast.RefStateHosted
	return ref
}

// createFoundReference builds a reference for a found zettel.
func (wui *WebUI) createFoundReference(zid id.Zid, fragment string) *ast.Reference {
	ub := wui.NewURLBuilder('h').SetZid(api.ZettelID(zid.String()))
	if fragment != "" {
		ub.SetFragment(fragment)
	}

	ref := ast.ParseReference(ub.String())
	ref.State = ast.RefStateFound
	return ref
}

func (wui *WebUI) createImageMaterial(zid id.Zid) *ast.EmbedRefNode {
	ub := wui.NewURLBuilder('z').SetZid(api.ZettelID(zid.String()))
	ref := ast.ParseReference(ub.String())
	ref.State = ast.RefStateFound
	return &ast.EmbedRefNode{Ref: ref}
}
