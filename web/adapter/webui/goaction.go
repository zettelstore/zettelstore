//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
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

	"zettelstore.de/z/usecase"
)

// MakeGetGoActionHandler creates a new HTTP handler to execute certain commands.
func (wui *WebUI) MakeGetGoActionHandler(ucRefresh usecase.Refresh) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Currently, command "refresh" is the only command to be executed.
		err := ucRefresh.Run(ctx)
		if err != nil {
			wui.reportError(ctx, w, err)
			return
		}
		wui.redirectFound(w, r, wui.NewURLBuilder('/'))
	}
}
