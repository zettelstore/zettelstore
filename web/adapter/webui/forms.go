//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package webui provides web-UI handlers for web requests.
package webui

import (
	"net/http"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
)

type formZettelData struct {
	Heading       string
	MetaTitle     string
	MetaRole      string
	MetaTags      string
	MetaSyntax    string
	MetaPairsRest []meta.Pair
	IsTextContent bool
	Content       string
}

func parseZettelForm(r *http.Request, zid id.Zid) (domain.Zettel, bool, error) {
	err := r.ParseForm()
	if err != nil {
		return domain.Zettel{}, false, err
	}

	var m *meta.Meta
	if postMeta, ok := trimmedFormValue(r, "meta"); ok {
		m = meta.NewFromInput(zid, input.NewInput([]byte(postMeta)))
	} else {
		m = meta.New(zid)
	}
	if postTitle, ok := trimmedFormValue(r, "title"); ok {
		m.Set(api.KeyTitle, postTitle)
	}
	if postTags, ok := trimmedFormValue(r, "tags"); ok {
		if tags := strings.Fields(postTags); len(tags) > 0 {
			m.SetList(api.KeyTags, tags)
		}
	}
	if postRole, ok := trimmedFormValue(r, "role"); ok {
		m.Set(api.KeyRole, postRole)
	}
	if postSyntax, ok := trimmedFormValue(r, "syntax"); ok {
		m.Set(api.KeySyntax, postSyntax)
	}
	if values, ok := r.PostForm["content"]; ok && len(values) > 0 {
		return domain.Zettel{
			Meta: m,
			Content: domain.NewContent(
				[]byte(strings.ReplaceAll(strings.TrimSpace(values[0]), "\r\n", "\n"))),
		}, true, nil
	}
	return domain.Zettel{
		Meta:    m,
		Content: domain.NewContent(nil),
	}, false, nil
}

func trimmedFormValue(r *http.Request, key string) (string, bool) {
	if values, ok := r.PostForm[key]; ok && len(values) > 0 {
		value := strings.TrimSpace(values[0])
		if len(value) > 0 {
			return value, true
		}
	}
	return "", false
}
