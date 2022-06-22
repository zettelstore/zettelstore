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
	"strconv"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/c/maps"
)

func (s *Search) String() string {
	var sb strings.Builder
	s.Print(&sb)
	return sb.String()
}

// Print the search to a writer.
func (s *Search) Print(w io.Writer) {
	if s == nil {
		return
	}
	if s.negate {
		io.WriteString(w, "NOT (")
	}
	space := false
	if len(s.search) > 0 {
		io.WriteString(w, "ANY")
		printSelectExprValues(w, s.search)
		space = true
	}
	for _, name := range maps.Keys(s.mvals) {
		if space {
			io.WriteString(w, " AND ")
		}
		io.WriteString(w, name)
		printSelectExprValues(w, s.mvals[name])
		space = true
	}
	if s.negate {
		io.WriteString(w, ")")
		space = true
	}

	space = printOrder(w, s.order, s.descending, space)
	space = printPosInt(w, "OFFSET", s.offset, space)
	_ = printPosInt(w, "LIMIT", s.limit, space)
}

func printSelectExprValues(w io.Writer, values []expValue) {
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
			io.WriteString(w, " EQUAL ")
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

func printOrder(w io.Writer, order string, descending, withSpace bool) bool {
	if len(order) > 0 {
		switch order {
		case api.KeyID:
			// Ignore
		case RandomOrder:
			withSpace = printSpace(w, withSpace)
			io.WriteString(w, "RANDOM")
		default:
			withSpace = printSpace(w, withSpace)
			io.WriteString(w, "SORT ")
			io.WriteString(w, order)
			if descending {
				io.WriteString(w, " DESC")
			}
		}
	}
	return withSpace
}

func printPosInt(w io.Writer, key string, val int, space bool) bool {
	if val > 0 {
		space = printSpace(w, space)
		io.WriteString(w, key)
		w.Write(bsSpace)
		io.WriteString(w, strconv.Itoa(val))
	}
	return space
}

var bsSpace = []byte{' '}

func printSpace(w io.Writer, space bool) bool {
	if space {
		w.Write(bsSpace)
	}
	return true
}
