//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package adapter provides handlers for web requests.
package adapter

import (
	"fmt"
	"net/http"

	"zettelstore.de/z/place"
	"zettelstore.de/z/usecase"
)

// ReportUsecaseError returns an appropriate HTTP status code for errors in use cases.
func ReportUsecaseError(w http.ResponseWriter, err error) {
	if err == place.ErrNotFound {
		NotFound(w, http.StatusText(404))
		return
	}
	if err, ok := err.(*place.ErrNotAllowed); ok {
		Forbidden(w, err.Error())
		return
	}
	if err, ok := err.(*place.ErrInvalidID); ok {
		BadRequest(w, fmt.Sprintf("Zettel-ID %q not appropriate in this context.", err.Zid.String()))
		return
	}
	if err, ok := err.(*usecase.ErrZidInUse); ok {
		BadRequest(w, fmt.Sprintf("Zettel-ID %q already in use.", err.Zid.String()))
		return
	}
	if err == place.ErrStopped {
		InternalServerError(w, "Zettelstore not operational.", err)
		return
	}
	InternalServerError(w, "", err)
}
