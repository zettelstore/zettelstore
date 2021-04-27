//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package search provides a zettel search.
package search

import (
	"io"
	"sort"
	"strconv"

	"zettelstore.de/z/domain/meta"
)

// Print the filter to a writer.
func (s *Search) Print(w io.Writer) {
	if s.negate {
		io.WriteString(w, "NOT (")
	}
	space := false
	if len(s.search) > 0 {
		io.WriteString(w, "ANY")
		printFilterExprValues(w, s.search)
		space = true
	}
	names := make([]string, 0, len(s.tags))
	for name := range s.tags {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if space {
			io.WriteString(w, " AND ")
		}
		io.WriteString(w, name)
		printFilterExprValues(w, s.tags[name])
		space = true
	}
	if s.negate {
		io.WriteString(w, ")")
		space = true
	}

	if ord := s.order; len(ord) > 0 {
		switch ord {
		case meta.KeyID:
			// Ignore
		case RandomOrder:
			space = printSpace(w, space)
			io.WriteString(w, "RANDOM")
		default:
			space = printSpace(w, space)
			io.WriteString(w, "SORT ")
			io.WriteString(w, ord)
			if s.descending {
				io.WriteString(w, " DESC")
			}
		}
	}
	if off := s.offset; off > 0 {
		space = printSpace(w, space)
		io.WriteString(w, "OFFSET ")
		io.WriteString(w, strconv.Itoa(off))
	}
	if lim := s.limit; lim > 0 {
		_ = printSpace(w, space)
		io.WriteString(w, "LIMIT ")
		io.WriteString(w, strconv.Itoa(lim))
	}
}

func printFilterExprValues(w io.Writer, values []expValue) {
	if len(values) == 0 {
		io.WriteString(w, " MATCH ANY")
		return
	}

	for j, val := range values {
		if j > 0 {
			io.WriteString(w, " AND")
		}
		if val.negate {
			io.WriteString(w, " NOT")
		}
		switch val.op {
		case cmpDefault:
			io.WriteString(w, " MATCH ")
		case cmpEqual:
			io.WriteString(w, " HAS ")
		case cmpPrefix:
			io.WriteString(w, " PREFIX ")
		case cmpSuffix:
			io.WriteString(w, " SUFFIX ")
		case cmpContains:
			io.WriteString(w, " CONTAINS ")
		default:
			io.WriteString(w, " MaTcH ")
		}
		if val.value == "" {
			io.WriteString(w, "ANY")
		} else {
			io.WriteString(w, val.value)
		}
	}
}

func printSpace(w io.Writer, space bool) bool {
	if space {
		io.WriteString(w, " ")
	}
	return true
}
