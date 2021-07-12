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
	"zettelstore.de/z/encoder"
	"zettelstore.de/z/search"
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

// GetFormat returns the data format selected by the caller.
func GetFormat(r *http.Request, q url.Values, defFormat encoder.Enum) (encoder.Enum, string) {
	format := q.Get(api.QueryKeyFormat)
	if len(format) > 0 {
		return api.Encoder(format), format
	}
	if format, ok := getOneFormat(r, api.HeaderAccept); ok {
		return api.Encoder(format), format
	}
	if format, ok := getOneFormat(r, api.HeaderContentType); ok {
		return api.Encoder(format), format
	}
	return defFormat, "*default*"
}

func getOneFormat(r *http.Request, key string) (string, bool) {
	if values, ok := r.Header[key]; ok {
		for _, value := range values {
			if format, ok := contentType2format(value); ok {
				return format, true
			}
		}
	}
	return "", false
}

var mapCT2format = map[string]string{
	"application/json": api.FormatJSON,
	"text/html":        api.FormatHTML,
}

func contentType2format(contentType string) (string, bool) {
	// TODO: only check before first ';'
	format, ok := mapCT2format[contentType]
	return format, ok
}

// GetSearch retrieves the specified filter and sorting options from a query.
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
