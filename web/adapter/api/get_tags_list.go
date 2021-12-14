//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
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
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeListTagsHandler creates a new HTTP handler for the use case "list some zettel".
func (a *API) MakeListTagsHandler(listTags usecase.ListTags) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		iMinCount, _ := strconv.Atoi(r.URL.Query().Get("min"))
		tagData, err := listTags.Run(r.Context(), iMinCount)
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}

		tagMap := make(map[string][]api.ZettelID, len(tagData))
		for tag, metaList := range tagData {
			zidList := make([]api.ZettelID, 0, len(metaList))
			for _, m := range metaList {
				zidList = append(zidList, api.ZettelID(m.Zid.String()))
			}
			tagMap[tag] = zidList
		}

		var buf bytes.Buffer
		err = encodeJSONData(&buf, api.TagListJSON{Tags: tagMap})
		if err != nil {
			a.log.Fatal().Err(err).Msg("Unable to store tag list in buffer")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		adapter.PrepareHeader(w, ctJSON)
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(buf.Bytes())
		a.log.IfErr(err).Msg("Write Tags")
	}
}
