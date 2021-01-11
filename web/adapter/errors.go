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
	"log"
	"net/http"
)

// BadRequest signals HTTP status code 400.
func BadRequest(w http.ResponseWriter, text string) {
	http.Error(w, text, http.StatusBadRequest)
}

// Forbidden signals HTTP status code 403.
func Forbidden(w http.ResponseWriter, text string) {
	http.Error(w, text, http.StatusForbidden)
}

// NotFound signals HTTP status code 404.
func NotFound(w http.ResponseWriter, text string) {
	http.Error(w, text, http.StatusNotFound)
}

// InternalServerError signals HTTP status code 500.
func InternalServerError(w http.ResponseWriter, text string, err error) {
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	if text == "" {
		log.Println(err)
	} else {
		log.Printf("%v: %v", text, err)
	}
}

// NotImplemented signals HTTP status code 501
func NotImplemented(w http.ResponseWriter, text string) {
	http.Error(w, text, http.StatusNotImplemented)
	log.Println(text)
}
