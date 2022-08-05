//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package search provides a zettel search.
package search

import (
	"bytes"
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

// Print the search in a parseable form.
func (s *Search) Print(w io.Writer) {
	if s == nil {
		return
	}
	env := printEnv{w: w}
	if s.negate {
		io.WriteString(w, "NEGATE")
		env.space = true
	}
	if len(s.search) > 0 {
		env.printExprValues("", s.search)
	}
	for _, name := range maps.Keys(s.mvals) {
		env.printExprValues(name, s.mvals[name])
	}
}

type printEnv struct {
	w     io.Writer
	space bool
}

func (pe *printEnv) printSpace() {
	if pe.space {
		pe.w.Write(bsSpace)
		return
	}
	pe.space = true
}
func (pe *printEnv) writeString(s string) { io.WriteString(pe.w, s) }
func (pe *printEnv) writeQuoted(s string) {
	var buf bytes.Buffer
	buf.WriteByte('"')
	for _, ch := range s {
		switch ch {
		case '\t':
			buf.WriteString("\\t")
		case '\r':
			buf.WriteString("\\r")
		case '\n':
			buf.WriteString("\\n")
		case '"':
			buf.WriteString("\\\"")
		case '\\':
			buf.WriteString("\\\\")
		default:
			if ch < ' ' {
				buf.WriteString("\\ ")
			} else {
				buf.WriteRune(ch)
			}
		}
	}
	buf.WriteByte('"')
	pe.w.Write(buf.Bytes())
}

func (pe *printEnv) printExprValues(key string, values []expValue) {
	for _, val := range values {
		pe.printSpace()
		pe.writeString(key)
		if val.negate {
			pe.writeString("!")
		}
		switch val.op {
		case cmpDefault:
			pe.writeString(":")
		case cmpEqual:
			pe.writeString("=")
		case cmpPrefix:
			pe.writeString(">")
		case cmpSuffix:
			pe.writeString("<")
		case cmpContains:
			// An empty key signals a full-text search. Since "~" is the default op in this case,
			// it can be ignored. Therefore, print only "~" if there is a key.
			if key != "" {
				pe.writeString("~")
			}
		}
		if s := val.value; s != "" {
			if needsQuote(s) {
				pe.writeQuoted(s)
			} else {
				pe.writeString(s)
			}
		}
	}
}

func needsQuote(s string) bool {
	for _, ch := range s {
		if ch == '\\' || ch == '"' || isSpace(ch) {
			return true
		}
	}
	return false
}

func (s *Search) Human() string {
	var sb strings.Builder
	s.PrintHuman(&sb)
	return sb.String()
}

// PrintHuman the search to a writer in a human readable form.
func (s *Search) PrintHuman(w io.Writer) {
	if s == nil {
		return
	}
	if s.negate {
		io.WriteString(w, "NOT (")
	}
	space := false
	if len(s.search) > 0 {
		io.WriteString(w, "ANY")
		printHumanSelectExprValues(w, s.search)
		space = true
	}
	for _, name := range maps.Keys(s.mvals) {
		if space {
			io.WriteString(w, " AND ")
		}
		io.WriteString(w, name)
		printHumanSelectExprValues(w, s.mvals[name])
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

func printHumanSelectExprValues(w io.Writer, values []expValue) {
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
