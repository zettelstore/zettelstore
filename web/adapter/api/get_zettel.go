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

	"zettelstore.de/c/api"
	"zettelstore.de/z/box"
	"zettelstore.de/z/config"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeGetZettelHandler creates a new HTTP handler to return a zettel.
func (a *API) MakeGetZettelHandler(getZettel usecase.GetZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		z, err := a.getZettelFromPath(r.Context(), w, r, getZettel)
		if err != nil {
			return
		}

		adapter.PrepareHeader(w, ctJSON)
		w.WriteHeader(http.StatusOK)
		content, encoding := z.Content.Encode()
		err = encodeJSONData(w, api.ZettelJSON{
			ID:       api.ZettelID(z.Meta.Zid.String()),
			Meta:     z.Meta.Map(),
			Encoding: encoding,
			Content:  content,
		})
		a.log.IfErr(err).Msg("Write JSON Zettel")
	}
}

// MakeGetPlainZettelHandler creates a new HTTP handler to return a zettel in plain formar
func (a *API) MakeGetPlainZettelHandler(getZettel usecase.GetZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		z, err := a.getZettelFromPath(box.NoEnrichContext(r.Context()), w, r, getZettel)
		if err != nil {
			return
		}

		switch getPart(r.URL.Query(), partContent) {
		case partZettel:
			w.WriteHeader(http.StatusOK)
			_, err = z.Meta.Write(w, false)
			if err == nil {
				_, err = w.Write([]byte{'\n'})
			}
			if err == nil {
				_, err = z.Content.Write(w)
			}
		case partMeta:
			adapter.PrepareHeader(w, ctPlainText)
			w.WriteHeader(http.StatusOK)
			_, err = z.Meta.Write(w, false)
		case partContent:
			if ct, ok := syntax2contentType(config.GetSyntax(z.Meta, a.rtConfig)); ok {
				adapter.PrepareHeader(w, ct)
			}
			w.WriteHeader(http.StatusOK)
			_, err = z.Content.Write(w)
		}
		a.log.IfErr(err).Zid(z.Meta.Zid).Msg("Write Plain Zettel")
	}
}

func (a *API) getZettelFromPath(ctx context.Context, w http.ResponseWriter, r *http.Request, getZettel usecase.GetZettel) (domain.Zettel, error) {
	zid, err := id.Parse(r.URL.Path[1:])
	if err != nil {
		http.NotFound(w, r)
		return domain.Zettel{}, err
	}

	z, err := getZettel.Run(ctx, zid)
	if err != nil {
		a.reportUsecaseError(w, err)
		return domain.Zettel{}, err
	}
	return z, nil
}

// MakeGetMetaHandler creates a new HTTP handler to return metadata of a zettel.
func (a *API) MakeGetMetaHandler(getMeta usecase.GetMeta) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}

		m, err := getMeta.Run(r.Context(), zid)
		if err != nil {
			a.reportUsecaseError(w, err)
			return
		}

		adapter.PrepareHeader(w, ctJSON)
		w.WriteHeader(http.StatusOK)
		err = encodeJSONData(w, api.MetaJSON{
			Meta: m.Map(),
		})
		a.log.IfErr(err).Zid(zid).Msg("Write JSON Meta")
	}
}
