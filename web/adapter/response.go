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
	"errors"
	"fmt"
	"log"
	"net/http"

	"zettelstore.de/z/box"
	"zettelstore.de/z/usecase"
)

// ReportUsecaseError returns an appropriate HTTP status code for errors in use cases.
func ReportUsecaseError(w http.ResponseWriter, err error) {
	code, text := CodeMessageFromError(err)
	if code == http.StatusInternalServerError {
		log.Printf("%v: %v", text, err)
	}
	http.Error(w, text, code)
}

// ErrBadRequest is returned if the caller made an invalid HTTP request.
type ErrBadRequest struct {
	Text string
}

// NewErrBadRequest creates an new bad request error.
func NewErrBadRequest(text string) error { return &ErrBadRequest{Text: text} }

func (err *ErrBadRequest) Error() string { return err.Text }

// CodeMessageFromError returns an appropriate HTTP status code and text from a given error.
func CodeMessageFromError(err error) (int, string) {
	if err == box.ErrNotFound {
		return http.StatusNotFound, http.StatusText(http.StatusNotFound)
	}
	if err1, ok := err.(*box.ErrNotAllowed); ok {
		return http.StatusForbidden, err1.Error()
	}
	if err1, ok := err.(*box.ErrInvalidID); ok {
		return http.StatusBadRequest, fmt.Sprintf("Zettel-ID %q not appropriate in this context", err1.Zid)
	}
	if err1, ok := err.(*usecase.ErrZidInUse); ok {
		return http.StatusBadRequest, fmt.Sprintf("Zettel-ID %q already in use", err1.Zid)
	}
	if err1, ok := err.(*ErrBadRequest); ok {
		return http.StatusBadRequest, err1.Text
	}
	if errors.Is(err, box.ErrStopped) {
		return http.StatusInternalServerError, fmt.Sprintf("Zettelstore not operational: %v", err)
	}
	if errors.Is(err, box.ErrConflict) {
		return http.StatusConflict, "Zettelstore operations conflicted"
	}
	return http.StatusInternalServerError, err.Error()
}
