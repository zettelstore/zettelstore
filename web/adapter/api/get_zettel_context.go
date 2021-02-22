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

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

type jsonMetaList struct {
	ID   string            `json:"id"`
	URL  string            `json:"url"`
	Meta map[string]string `json:"meta"`
	List []jsonMeta        `json:"list"`
}

// MakeZettelContextHandler creates a new HTTP handler for the use case "zettel context".
func MakeZettelContextHandler(getContext usecase.ZettelContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		dir := usecase.ParseZCDirection(q.Get("dir"))
		depth, ok := adapter.GetInteger(q, "depth")
		if !ok || depth < 0 {
			depth = 5
		}
		limit, ok := adapter.GetInteger(q, "limit")
		if !ok || limit < 0 {
			limit = 200
		}
		ctx := r.Context()
		metaList, err := getContext.Run(ctx, zid, dir, depth, limit)
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}
		outData := jsonMetaList{
			ID:   metaList[0].Zid.String(),
			URL:  adapter.NewURLBuilder('z').SetZid(zid).String(),
			Meta: metaList[0].Map(),
			List: make([]jsonMeta, len(metaList)-1),
		}
		for i, m := range metaList[1:] {
			outData.List[i].ID = m.Zid.String()
			outData.List[i].URL = adapter.NewURLBuilder('z').SetZid(m.Zid).String()
			outData.List[i].Meta = m.Map()
		}
		encodeJSONData(w, outData, true)
	}
}
