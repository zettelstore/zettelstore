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

	zsapi "zettelstore.de/z/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetZettelHandler creates a new HTTP handler to return a rendered zettel.
func (api *API) MakeGetZettelHandler(getZettel usecase.GetZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		q := r.URL.Query()
		_, isRaw := q[zsapi.QueryKeyRaw]
		if isRaw {
			ctx = box.NoEnrichContext(ctx)
		}
		z, err := getZettel.Run(ctx, zid)
		if err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}

		if isRaw {
			part := getPart(q, partContent)
			if part == partUnknown {
				part = partContent
			}
			api.writeRawZettel(w, z, part)
			return
		}

		w.Header().Set(zsapi.HeaderContentType, ctJSON)
		content, encoding := z.Content.Encode()
		err = encodeJSONData(w, zsapi.ZettelJSON{
			ID:       zid.String(),
			URL:      api.NewURLBuilder('z').SetZid(zid).String(),
			Meta:     z.Meta.Map(),
			Encoding: encoding,
			Content:  content,
		})
		if err != nil {
			adapter.InternalServerError(w, "Write Zettel JSON", err)
		}
	}
}

func (api *API) writeRawZettel(w http.ResponseWriter, z domain.Zettel, part partType) {
	var err error
	switch part {
	case partZettel:
		_, err = z.Meta.Write(w, false)
		if err == nil {
			_, err = w.Write([]byte{'\n'})
		}
		if err == nil {
			_, err = z.Content.Write(w)
		}
	case partMeta:
		w.Header().Set(zsapi.HeaderContentType, ctPlainText)
		_, err = z.Meta.Write(w, false)
	case partContent:
		if ct, ok := syntax2contentType(config.GetSyntax(z.Meta, api.rtConfig)); ok {
			w.Header().Set(zsapi.HeaderContentType, ct)
		}
		_, err = z.Content.Write(w)
	}
	if err != nil {
		adapter.InternalServerError(w, "Write raw zettel", err)
	}
}
