//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
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
)

func (wui *WebUI) redirectFound(w http.ResponseWriter, r *http.Request, ub *api.URLBuilder) {
	us := ub.String()
	wui.log.Debug().Str("uri", us).Msg("redirect")
	http.Redirect(w, r, us, http.StatusFound)
}
