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
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/search"
)

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

// ContentType defines the HTTP header value "Content-Type".
const ContentType = "Content-Type"

// GetFormat returns the data format selected by the caller.
func GetFormat(r *http.Request, q url.Values, defFormat string) string {
	format := q.Get("_format")
	if len(format) > 0 {
		return format
	}
	if format, ok := getOneFormat(r, "Accept"); ok {
		return format
	}
	if format, ok := getOneFormat(r, ContentType); ok {
		return format
	}
	return defFormat
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
	"application/json": "json",
	"text/html":        "html",
}

func contentType2format(contentType string) (string, bool) {
	// TODO: only check before first ';'
	format, ok := mapCT2format[contentType]
	return format, ok
}

// GetFilterSorter retrieves the specified filter and sorting options from a query.
func GetFilterSorter(q url.Values, forSearch bool) (filter *search.Filter, sorter *search.Sorter) {
	sortQKey, orderQKey, offsetQKey, limitQKey, negateQKey, sQKey := getQueryKeys(forSearch)
	for key, values := range q {
		switch key {
		case sortQKey, orderQKey:
			sorter = extractOrderFromQuery(values, sorter)
		case offsetQKey:
			sorter = extractOffsetFromQuery(values, sorter)
		case limitQKey:
			sorter = extractLimitFromQuery(values, sorter)
		case negateQKey:
			filter = filter.SetNegate()
		case sQKey:
			filter = setCleanedQueryValues(filter, "", values)
		default:
			if !forSearch && meta.KeyIsValid(key) {
				filter = setCleanedQueryValues(filter, key, values)
			}
		}
	}
	return filter, sorter
}

func extractOrderFromQuery(values []string, sorter *search.Sorter) *search.Sorter {
	if len(values) > 0 {
		descending := false
		sortkey := values[0]
		if strings.HasPrefix(sortkey, "-") {
			descending = true
			sortkey = sortkey[1:]
		}
		if meta.KeyIsValid(sortkey) || sortkey == search.RandomOrder {
			sorter = sorter.Ensure()
			sorter.Order = sortkey
			sorter.Descending = descending
		}
	}
	return sorter
}

func extractOffsetFromQuery(values []string, sorter *search.Sorter) *search.Sorter {
	if len(values) > 0 {
		if offset, err := strconv.Atoi(values[0]); err == nil {
			sorter = sorter.Ensure()
			sorter.Offset = offset
		}
	}
	return sorter
}

func extractLimitFromQuery(values []string, sorter *search.Sorter) *search.Sorter {
	if len(values) > 0 {
		if limit, err := strconv.Atoi(values[0]); err == nil {
			sorter = sorter.Ensure()
			sorter.Limit = limit
		}
	}
	return sorter
}

func getQueryKeys(forSearch bool) (string, string, string, string, string, string) {
	if forSearch {
		return "sort", "order", "offset", "limit", "negate", "s"
	}
	return "_sort", "_order", "_offset", "_limit", "_negate", "_s"
}

func setCleanedQueryValues(filter *search.Filter, key string, values []string) *search.Filter {
	for _, val := range values {
		val = strings.TrimSpace(val)
		if len(val) > 0 && val[0] == '!' {
			filter = filter.AddExpr(key, val[1:], true)
		} else {
			filter = filter.AddExpr(key, val, false)
		}
	}
	return filter
}
