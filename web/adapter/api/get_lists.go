//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package api provides api handlers for web requests.
package api

import (
	"bytes"
	"net/http"
	"strconv"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/usecase"
)

// MakeListRoleHandler creates a new HTTP handler for the use case "list roles".
func (a *API) MakeListRoleHandler(listRole usecase.ListRoles) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleArrangement, err := listRole.Run(r.Context())
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}
		if err2, wrote := a.writeArrangement(w, roleArrangement); wrote {
			a.log.IfErr(err2).Msg("Write Roles")
		}
	}
}

// MakeListTagsHandler creates a new HTTP handler for the use case "list some zettel".
func (a *API) MakeListTagsHandler(listTags usecase.ListTags) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		iMinCount, err := strconv.Atoi(r.URL.Query().Get("min"))
		if err != nil || iMinCount < 0 {
			iMinCount = 0
		}
		tagData, err := listTags.Run(r.Context(), iMinCount)
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}

		if err2, wrote := a.writeArrangement(w, tagData); wrote {
			a.log.IfErr(err2).Msg("Write Tags")
		}
	}
}

func (a *API) writeArrangement(w http.ResponseWriter, ar meta.Arrangement) (error, bool) {
	mm := make(api.MapMeta, len(ar))
	for tag, metaList := range ar {
		zidList := make([]api.ZettelID, 0, len(metaList))
		for _, m := range metaList {
			zidList = append(zidList, api.ZettelID(m.Zid.String()))
		}
		mm[tag] = zidList
	}

	var buf bytes.Buffer
	err := encodeJSONData(&buf, api.MapListJSON{Map: mm})
	if err != nil {
		a.log.Fatal().Err(err).Msg("Unable to store map list in buffer")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return nil, false
	}

	err = writeBuffer(w, &buf, ctJSON)
	return err, true
}
