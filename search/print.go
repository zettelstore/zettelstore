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

var bsSpace = []byte{' '}

func (pe *printEnv) printSpace() {
	if pe.space {
		pe.w.Write(bsSpace)
		return
	}
	pe.space = true
}
func (pe *printEnv) writeString(s string) { io.WriteString(pe.w, s) }

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
			pe.writeString(s)
		}
	}
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
	env := printEnv{w: w}
	if s.negate {
		env.writeString("NOT (")
	}
	if len(s.search) > 0 {
		env.writeString("ANY")
		env.printHumanSelectExprValues(s.search)
		env.space = true
	}
	for _, name := range maps.Keys(s.mvals) {
		if env.space {
			env.writeString(" AND ")
		}
		env.writeString(name)
		env.printHumanSelectExprValues(s.mvals[name])
		env.space = true
	}
	if s.negate {
		env.writeString(")")
		env.space = true
	}

	env.printOrder(s.order, s.descending)
	env.printPosInt("OFFSET", s.offset)
	env.printPosInt("LIMIT", s.limit)
}

func (pe *printEnv) printHumanSelectExprValues(values []expValue) {
	if len(values) == 0 {
		pe.writeString(" MATCH ANY")
		return
	}

	for j, val := range values {
		if j > 0 {
			pe.writeString(" AND")
		}
		if val.negate {
			pe.writeString(" NOT")
		}
		switch val.op {
		case cmpDefault:
			pe.writeString(" MATCH ")
		case cmpEqual:
			pe.writeString(" EQUAL ")
		case cmpPrefix:
			pe.writeString(" PREFIX ")
		case cmpSuffix:
			pe.writeString(" SUFFIX ")
		case cmpContains:
			pe.writeString(" CONTAINS ")
		default:
			pe.writeString(" MaTcH ")
		}
		if val.value == "" {
			pe.writeString("ANY")
		} else {
			pe.writeString(val.value)
		}
	}
}

func (pe *printEnv) printOrder(order string, descending bool) {
	if len(order) > 0 {
		switch order {
		case api.KeyID:
			// Ignore
		case RandomOrder:
			pe.printSpace()
			pe.writeString("RANDOM")
		default:
			pe.printSpace()
			pe.writeString("SORT ")
			pe.writeString(order)
			if descending {
				pe.writeString(" DESC")
			}
		}
	}
}

func (pe *printEnv) printPosInt(key string, val int) {
	if val > 0 {
		pe.printSpace()
		pe.writeString(key)
		pe.writeString(" ")
		pe.writeString(strconv.Itoa(val))
	}
}
