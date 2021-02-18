//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
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
	"encoding/json"
	"net/http"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

type jsonGetOrder struct {
	ID    string      `json:"id"`
	URL   string      `json:"url"`
	Order []jsonIDURL `json:"order"`
}

// MakeGetOrderHandler creates a new API handler to return zettel references
// of a given zettel.
func MakeGetOrderHandler(zettelOrder usecase.ZettelOrder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		metas, err := zettelOrder.Run(r.Context(), zid, q.Get("syntax"))
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}
		outData := jsonGetOrder{
			ID:    zid.String(),
			URL:   adapter.NewURLBuilder('z').SetZid(zid).String(),
			Order: make([]jsonIDURL, 0, len(metas)),
		}
		for _, m := range metas {
			outData.Order = append(
				outData.Order,
				jsonIDURL{
					ID:  m.Zid.String(),
					URL: adapter.NewURLBuilder('z').SetZid(m.Zid).String(),
				},
			)
		}
		w.Header().Set("Content-Type", format2ContentType("json"))
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		enc.Encode(&outData)
	}
}
