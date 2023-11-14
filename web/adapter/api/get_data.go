//-----------------------------------------------------------------------------
// Copyright (c) 2022-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package api

import (
	"net/http"

	"zettelstore.de/sx.fossil"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/zettel/id"
)

// MakeGetDataHandler creates a new HTTP handler to return zettelstore data.
func (a *API) MakeGetDataHandler(ucVersion usecase.Version) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		version := ucVersion.Run()
		err := a.writeObject(w, id.Invalid, sx.MakeList(
			sx.Int64(version.Major),
			sx.Int64(version.Minor),
			sx.Int64(version.Patch),
			sx.String(version.Info),
			sx.String(version.Hash),
		))
		if err != nil {
			a.log.Error().Err(err).Msg("Write Version Info")
		}
	}
}
