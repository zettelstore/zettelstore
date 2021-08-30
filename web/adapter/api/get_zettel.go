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
	"context"
	"net/http"

	zsapi "zettelstore.de/z/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetZettelHandler creates a new HTTP handler to return a zettel.
func MakeGetZettelHandler(getZettel usecase.GetZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		z, err := getZettelFromPath(r.Context(), w, r, getZettel)
		if err != nil {
			return
		}

		w.Header().Set(zsapi.HeaderContentType, ctJSON)
		content, encoding := z.Content.Encode()
		err = encodeJSONData(w, zsapi.ZettelJSON{
			ID:       z.Meta.Zid.String(),
			Meta:     z.Meta.Map(),
			Encoding: encoding,
			Content:  content,
		})
		if err != nil {
			adapter.InternalServerError(w, "Write Zettel JSON", err)
		}
	}
}

// MakeGetPlainZettelHandler creates a new HTTP handler to return a zettel in plain formar
func (a *API) MakeGetPlainZettelHandler(getZettel usecase.GetZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		z, err := getZettelFromPath(box.NoEnrichContext(r.Context()), w, r, getZettel)
		if err != nil {
			return
		}

		switch getPart(r.URL.Query(), partContent) {
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
			if ct, ok := syntax2contentType(config.GetSyntax(z.Meta, a.rtConfig)); ok {
				w.Header().Set(zsapi.HeaderContentType, ct)
			}
			_, err = z.Content.Write(w)
		}
		if err != nil {
			adapter.InternalServerError(w, "Write plain zettel", err)
		}
	}
}

func getZettelFromPath(ctx context.Context, w http.ResponseWriter, r *http.Request, getZettel usecase.GetZettel) (domain.Zettel, error) {
	zid, err := id.Parse(r.URL.Path[1:])
	if err != nil {
		http.NotFound(w, r)
		return domain.Zettel{}, err
	}

	z, err := getZettel.Run(ctx, zid)
	if err != nil {
		adapter.ReportUsecaseError(w, err)
		return domain.Zettel{}, err
	}
	return z, nil
}
