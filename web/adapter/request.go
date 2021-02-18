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
	"zettelstore.de/z/place"
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

// GetFormat returns the data format selected by the caller.
func GetFormat(r *http.Request, q url.Values, defFormat string) string {
	format := q.Get("_format")
	if len(format) > 0 {
		return format
	}
	if format, ok := getOneFormat(r, "Accept"); ok {
		return format
	}
	if format, ok := getOneFormat(r, "Content-Type"); ok {
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
func GetFilterSorter(q url.Values, forSearch bool) (filter *place.Filter, sorter *place.Sorter) {
	sortQKey, orderQKey, offsetQKey, limitQKey, negateQKey, sQKey := getQueryKeys(forSearch)
	for key, values := range q {
		switch key {
		case sortQKey, orderQKey:
			if len(values) > 0 {
				descending := false
				sortkey := values[0]
				if strings.HasPrefix(sortkey, "-") {
					descending = true
					sortkey = sortkey[1:]
				}
				if meta.KeyIsValid(sortkey) || sortkey == place.RandomOrder {
					sorter = place.EnsureSorter(sorter)
					sorter.Order = sortkey
					sorter.Descending = descending
				}
			}
		case offsetQKey:
			if len(values) > 0 {
				if offset, err := strconv.Atoi(values[0]); err == nil {
					sorter = place.EnsureSorter(sorter)
					sorter.Offset = offset
				}
			}
		case limitQKey:
			if len(values) > 0 {
				if limit, err := strconv.Atoi(values[0]); err == nil {
					sorter = place.EnsureSorter(sorter)
					sorter.Limit = limit
				}
			}
		case negateQKey:
			filter = place.EnsureFilter(filter)
			filter.Negate = true
		case sQKey:
			if vals := cleanQueryValues(values); len(vals) > 0 {
				filter = place.EnsureFilter(filter)
				filter.Expr[""] = vals
			}
		default:
			if !forSearch && meta.KeyIsValid(key) {
				filter = place.EnsureFilter(filter)
				filter.Expr[key] = cleanQueryValues(values)
			}
		}
	}
	return filter, sorter
}

func getQueryKeys(forSearch bool) (string, string, string, string, string, string) {
	if forSearch {
		return "sort", "order", "offset", "limit", "negate", "s"
	}
	return "_sort", "_order", "_offset", "_limit", "_negate", "_s"
}

func cleanQueryValues(values []string) []string {
	result := make([]string, 0, len(values))
	for _, val := range values {
		val = strings.TrimSpace(val)
		if len(val) > 0 {
			result = append(result, val)
		}
	}
	return result
}
