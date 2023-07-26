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

	"zettelstore.de/sx.fossil/sxpf"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/zettel/id"
)

// MakeGetDataHandler creates a new HTTP handler to return zettelstore data.
func (a *API) MakeGetDataHandler(ucVersion usecase.Version) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		version := ucVersion.Run()
		err := a.writeObject(w, id.Invalid, sxpf.MakeList(
			sxpf.Int64(version.Major),
			sxpf.Int64(version.Minor),
			sxpf.Int64(version.Patch),
			sxpf.MakeString(version.Info),
			sxpf.MakeString(version.Hash),
		))
		a.log.IfErr(err).Msg("Write Version Info")
	}
}
