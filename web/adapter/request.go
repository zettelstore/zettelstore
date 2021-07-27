//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
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
	"net/url"
	"strconv"
	"strings"

	"zettelstore.de/z/api"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/search"
	"zettelstore.de/z/usecase"
)

// GetCredentialsViaForm retrieves the authentication credentions from a form.
func GetCredentialsViaForm(r *http.Request) (ident, cred string, ok bool) {
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		return "", "", false
	}

	ident = strings.TrimSpace(r.PostFormValue("username"))
	cred = r.PostFormValue("password")
	if ident == "" {
		return "", "", false
	}
	return ident, cred, true
}

// GetInteger returns the integer value of the named query key.
func GetInteger(q url.Values, key string) (int, bool) {
	s := q.Get(key)
	if s != "" {
		if val, err := strconv.Atoi(s); err == nil {
			return val, true
		}
	}
	return 0, false
}

// GetEncoding returns the data encoding selected by the caller.
func GetEncoding(r *http.Request, q url.Values, defEncoding api.EncodingEnum) (api.EncodingEnum, string) {
	encoding := q.Get(api.QueryKeyEncoding)
	if len(encoding) > 0 {
		return api.Encoder(encoding), encoding
	}
	if enc, ok := getOneEncoding(r, api.HeaderAccept); ok {
		return api.Encoder(enc), enc
	}
	if enc, ok := getOneEncoding(r, api.HeaderContentType); ok {
		return api.Encoder(enc), enc
	}
	return defEncoding, "*default*"
}

func getOneEncoding(r *http.Request, key string) (string, bool) {
	if values, ok := r.Header[key]; ok {
		for _, value := range values {
			if enc, ok := contentType2encoding(value); ok {
				return enc, true
			}
		}
	}
	return "", false
}

var mapCT2encoding = map[string]string{
	"application/json": "json",
	"text/html":        api.EncodingHTML,
}

func contentType2encoding(contentType string) (string, bool) {
	// TODO: only check before first ';'
	enc, ok := mapCT2encoding[contentType]
	return enc, ok
}

// GetSearch retrieves the specified search and sorting options from a query.
func GetSearch(q url.Values, forSearch bool) (s *search.Search) {
	sortQKey, orderQKey, offsetQKey, limitQKey, negateQKey, sQKey := getQueryKeys(forSearch)
	for key, values := range q {
		switch key {
		case sortQKey, orderQKey:
			s = extractOrderFromQuery(values, s)
		case offsetQKey:
			s = extractOffsetFromQuery(values, s)
		case limitQKey:
			s = extractLimitFromQuery(values, s)
		case negateQKey:
			s = s.SetNegate()
		case sQKey:
			s = setCleanedQueryValues(s, "", values)
		default:
			if !forSearch && meta.KeyIsValid(key) {
				s = setCleanedQueryValues(s, key, values)
			}
		}
	}
	return s
}

func extractOrderFromQuery(values []string, s *search.Search) *search.Search {
	if len(values) > 0 {
		descending := false
		sortkey := values[0]
		if strings.HasPrefix(sortkey, "-") {
			descending = true
			sortkey = sortkey[1:]
		}
		if meta.KeyIsValid(sortkey) || sortkey == search.RandomOrder {
			s = s.AddOrder(sortkey, descending)
		}
	}
	return s
}

func extractOffsetFromQuery(values []string, s *search.Search) *search.Search {
	if len(values) > 0 {
		if offset, err := strconv.Atoi(values[0]); err == nil {
			s = s.SetOffset(offset)
		}
	}
	return s
}

func extractLimitFromQuery(values []string, s *search.Search) *search.Search {
	if len(values) > 0 {
		if limit, err := strconv.Atoi(values[0]); err == nil {
			s = s.SetLimit(limit)
		}
	}
	return s
}

func getQueryKeys(forSearch bool) (string, string, string, string, string, string) {
	if forSearch {
		return "sort", "order", "offset", "limit", "negate", "s"
	}
	return "_sort", "_order", "_offset", "_limit", "_negate", "_s"
}

func setCleanedQueryValues(s *search.Search, key string, values []string) *search.Search {
	for _, val := range values {
		s = s.AddExpr(key, val)
	}
	return s
}

// GetZCDirection returns a direction value for a given string.
func GetZCDirection(s string) usecase.ZettelContextDirection {
	switch s {
	case api.DirBackward:
		return usecase.ZettelContextBackward
	case api.DirForward:
		return usecase.ZettelContextForward
	}
	return usecase.ZettelContextBoth
}
