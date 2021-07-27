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
	"net/http"
	"strconv"

	zsapi "zettelstore.de/z/api"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeListTagsHandler creates a new HTTP handler for the use case "list some zettel".
func (api *API) MakeListTagsHandler(listTags usecase.ListTags) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		iMinCount, _ := strconv.Atoi(r.URL.Query().Get("min"))
		tagData, err := listTags.Run(r.Context(), iMinCount)
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}

		w.Header().Set(zsapi.HeaderContentType, ctJSON)
		tagMap := make(map[string][]string, len(tagData))
		for tag, metaList := range tagData {
			zidList := make([]string, 0, len(metaList))
			for _, m := range metaList {
				zidList = append(zidList, m.Zid.String())
			}
			tagMap[tag] = zidList
		}
		encodeJSONData(w, zsapi.TagListJSON{Tags: tagMap})
	}
}
