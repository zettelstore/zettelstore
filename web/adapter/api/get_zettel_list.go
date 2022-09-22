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
	"fmt"
	"net/http"

	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeListMetaHandler creates a new HTTP handler for the use case "list some zettel".
func (a *API) MakeListMetaHandler(listMeta usecase.ListMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		q := adapter.GetQuery(r.URL.Query())
		metaList, err := listMeta.Run(ctx, q)
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}

		var buf bytes.Buffer
		if err = a.writeQueryMetaList(ctx, &buf, q, metaList); err != nil {
			a.log.Fatal().Err(err).Msg("Unable to store meta list in buffer")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		err = writeBuffer(w, &buf, ctJSON)
		a.log.IfErr(err).Msg("Write JSON List")
	}
}

// MakeListPlainHandler creates a new HTTP handler for the use case "list some zettel".
func (a *API) MakeListPlainHandler(listMeta usecase.ListMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		metaList, err := listMeta.Run(ctx, adapter.GetQuery(r.URL.Query()))
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}

		var buf bytes.Buffer
		for _, m := range metaList {
			_, err = fmt.Fprintln(&buf, m.Zid.String(), m.GetTitle())
			if err != nil {
				a.log.Fatal().Err(err).Msg("Unable to store plain list in buffer")
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		}

		err = writeBuffer(w, &buf, ctPlainText)
		a.log.IfErr(err).Msg("Write Plain List")
	}
}
