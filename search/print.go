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
func (f *Filter) Print(w io.Writer) {
	if f.negate {
		io.WriteString(w, "NOT (")
	}
	useAnd := false
	if len(f.search) > 0 {
		io.WriteString(w, "ANY")
		printFilterExprValues(w, f.search)
		useAnd = true
	}
	names := make([]string, 0, len(f.tags))
	for name := range f.tags {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if useAnd {
			io.WriteString(w, " AND ")
		}
		io.WriteString(w, name)
		printFilterExprValues(w, f.tags[name])
		useAnd = true
	}
	if f.negate {
		io.WriteString(w, ")")
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
		io.WriteString(w, " MATCH ")
		if val.value == "" {
			io.WriteString(w, "ANY")
		} else {
			io.WriteString(w, val.value)
		}
	}
}

// Print the sorter to a writer.
func (s *Sorter) Print(w io.Writer) {
	var space bool
	if ord := s.Order; len(ord) > 0 {
		switch ord {
		case meta.KeyID:
			// Ignore
		case RandomOrder:
			io.WriteString(w, "RANDOM")
			space = true
		default:
			io.WriteString(w, "SORT ")
			io.WriteString(w, ord)
			if s.Descending {
				io.WriteString(w, " DESC")
			}
			space = true
		}
	}
	if off := s.Offset; off > 0 {
		if space {
			io.WriteString(w, " ")
		}
		io.WriteString(w, "OFFSET ")
		io.WriteString(w, strconv.Itoa(off))
		space = true
	}
	if lim := s.Limit; lim > 0 {
		if space {
			io.WriteString(w, " ")
		}
		io.WriteString(w, "LIMIT ")
		io.WriteString(w, strconv.Itoa(lim))
	}
}
