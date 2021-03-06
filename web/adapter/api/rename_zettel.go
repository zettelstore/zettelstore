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
	"net/http"
	"net/url"

	"zettelstore.de/z/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/usecase"
	"zettelstore.de/z/web/adapter"
)

// MakeRenameZettelHandler creates a new HTTP handler to update a zettel.
func (api *API) MakeRenameZettelHandler(renameZettel usecase.RenameZettel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zid, err := id.Parse(r.URL.Path[1:])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		newZid, found := getDestinationZid(r)
		if !found {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if err := renameZettel.Run(r.Context(), zid, newZid); err != nil {
			adapter.ReportUsecaseError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func getDestinationZid(r *http.Request) (id.Zid, bool) {
	if values, ok := r.Header[api.HeaderDestination]; ok {
		for _, value := range values {
			if zid, ok := getZidFromURL(value); ok {
				return zid, true
			}
		}
	}
	return id.Invalid, false
}

var zidLength = len(id.VersionZid.Bytes())

func getZidFromURL(val string) (id.Zid, bool) {
	u, err := url.Parse(val)
	if err != nil {
		return id.Invalid, false
	}
	if len(u.Path) < zidLength {
		return id.Invalid, false
	}
	zid, err := id.Parse(u.Path[len(u.Path)-zidLength:])
	if err != nil {
		return id.Invalid, false
	}
	return zid, true
}
