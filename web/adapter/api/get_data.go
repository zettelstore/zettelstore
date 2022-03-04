//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package api

import (
	"bytes"
	"net/http"

	"zettelstore.de/c/api"
	"zettelstore.de/z/usecase"
)

// MakeGetDataHandler creates a new HTTP handler to return zettelstore data.
func (a *API) MakeGetDataHandler(ucVersion usecase.Version) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		version := ucVersion.Run()
		result := api.VersionJSON{
			Major: version.Major,
			Minor: version.Minor,
			Patch: version.Patch,
			Info:  version.Info,
			Hash:  version.Hash,
		}
		var buf bytes.Buffer
		err := encodeJSONData(&buf, result)
		if err != nil {
			a.log.Fatal().Err(err).Msg("Unable to version info in buffer")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		err = writeBuffer(w, &buf, ctJSON)
		a.log.IfErr(err).Msg("Write Version Info")
	}
}
