//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package api

import (
	"net/http"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeListUnlinkedMetaHandler creates a new HTTP handler for the use case "list unlinked references".
func MakeListUnlinkedMetaHandler(
	getMeta usecase.GetMeta,
	unlinkedRefs usecase.UnlinkedReferences,
	evaluate *usecase.Evaluate,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		ctx := r.Context()
		zm, err := getMeta.Run(ctx, zid)
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}

		q := r.URL.Query()
		phrase := q.Get("_phrase")
		if phrase == "" {
			zmkTitle := zm.GetDefault(api.KeyTitle, "")
			ilnTitle := evaluate.RunMetadata(ctx, zmkTitle, nil)
			encdr := encoder.Create(api.EncoderText, nil)
			var b strings.Builder
			_, err = encdr.WriteInlines(&b, ilnTitle)
			if err == nil {
				phrase = b.String()
			}
		}

		metaList, err := unlinkedRefs.Run(
			ctx, phrase, adapter.AddUnlinkedRefsToSearch(adapter.GetSearch(q), zm))
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}

		result := api.ZidMetaRelatedList{
			ID:   api.ZettelID(zid.String()),
			Meta: zm.Map(),
			List: make([]api.ZidMetaJSON, 0, len(metaList)),
		}
		for _, m := range metaList {
			result.List = append(result.List, api.ZidMetaJSON{
				ID:   api.ZettelID(m.Zid.String()),
				Meta: m.Map(),
			})
		}

		adapter.PrepareHeader(w, ctJSON)
		if err = encodeJSONData(w, result); err != nil {
			adapter.InternalServerError(w, "Write unlinked references JSON", err)
		}
	}
}
