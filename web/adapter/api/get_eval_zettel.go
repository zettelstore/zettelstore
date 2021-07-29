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
	"fmt"
	"net/http"

	zsapi "zettelstore.de/z/api"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetEvalZettelHandler creates a new HTTP handler to return a evaluated zettel.
func (api *API) MakeGetEvalZettelHandler(evaluateZettel usecase.EvaluateZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		q := r.URL.Query()
		enc, _ := adapter.GetEncoding(r, q, encoder.GetDefaultEncoding())
		part := getPart(q, partZettel)
		zn, err := evaluateZettel.Run(ctx, zid, &usecase.EvaluateEnvironment{
			Syntax:        q.Get(meta.KeySyntax),
			Encoding:      enc,
			Key:           'v',
			Part:          part.DefString(partZettel),
			GetURLPrefix:  api.GetURLPrefix,
			NewURLBuilder: api.NewURLBuilder,
		})
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}

		env := encoder.Environment{
			Lang:           config.GetLang(zn.InhMeta, api.rtConfig),
			Xhtml:          false,
			MarkerExternal: "",
			NewWindow:      false,
			IgnoreMeta:     map[string]bool{meta.KeyLang: true},
		}
		encdr := encoder.Create(enc, &env)
		if encdr == nil {
			adapter.BadRequest(w, fmt.Sprintf("Zettel %q not available in encoding %q", zid.String(), enc))
			return
		}
		w.Header().Set(zsapi.HeaderContentType, encoding2ContentType(enc))
		switch part {
		case partZettel:
			_, err = encdr.WriteZettel(w, zn)
		case partMeta:
			_, err = encdr.WriteMeta(w, zn.InhMeta)
		case partContent:
			_, err = encdr.WriteContent(w, zn)
		}
		if err != nil {
			adapter.InternalServerError(w, "Get evaluated zettel", err)
		}
	}
}
