//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
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
	"strconv"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/usecase"
)

// MakeListMapMetaHandler creates a new HTTP handler to retrieve mappings of
// metadata values of a specific key to the list of zettel IDs, which contain
// this value.
func (a *API) MakeListMapMetaHandler(listRole usecase.ListRoles, listTags usecase.ListTags) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var buf bytes.Buffer
		query := r.URL.Query()

		iMinCount, err := strconv.Atoi(query.Get(api.QueryKeyMin))
		if err != nil || iMinCount < 0 {
			iMinCount = 0
		}
		key := query.Get(api.QueryKeyKey)
		var ar meta.Arrangement
		switch key {
		case api.KeyRole:
			ar, err = listRole.Run(ctx)
		case api.KeyTags:
			ar, err = listTags.Run(ctx, iMinCount)
		default:
			a.log.Info().Str("key", key).Msg("illegal key for retrieving meta map")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}

		mm := make(api.MapMeta, len(ar))
		for tag, metaList := range ar {
			zidList := make([]api.ZettelID, 0, len(metaList))
			for _, m := range metaList {
				zidList = append(zidList, api.ZettelID(m.Zid.String()))
			}
			mm[tag] = zidList
		}

		buf.Reset()
		err = encodeJSONData(&buf, api.MapListJSON{Map: mm})
		if err != nil {
			a.log.Fatal().Err(err).Msg("Unable to store map list in buffer")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		err = writeBuffer(w, &buf, ctJSON)
		a.log.IfErr(err).Str("key", key).Msg("write meta map")
	}
}
