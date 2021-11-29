//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
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

func redirectFound(w http.ResponseWriter, r *http.Request, ub *api.URLBuilder) {
	http.Redirect(w, r, ub.String(), http.StatusFound)
}

func (wui *WebUI) createImageMaterial(zid id.Zid) ast.MaterialNode {
	ub := wui.NewURLBuilder('z').SetZid(api.ZettelID(zid.String()))
	ref := ast.ParseReference(ub.String())
	ref.State = ast.RefStateFound
	return &ast.ReferenceMaterialNode{Ref: ref}
}
