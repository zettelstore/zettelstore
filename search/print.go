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

var op2string = map[compareOp]string{
	cmpExist:    api.SearchOperatorExist,
	cmpNotExist: api.SearchOperatorNotExist,
	cmpHas:      api.SearchOperatorHas,
	cmpHasNot:   api.SearchOperatorHasNot,
	cmpPrefix:   api.SearchOperatorPrefix,
	cmpNoPrefix: api.SearchOperatorNoPrefix,
	cmpSuffix:   api.SearchOperatorSuffix,
	cmpNoSuffix: api.SearchOperatorNoSuffix,
	cmpMatch:    api.SearchOperatorMatch,
	cmpNoMatch:  api.SearchOperatorNoMatch,
}

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
		io.WriteString(w, kwNegate)
		env.space = true
	}
	for _, name := range maps.Keys(s.keyExist) {
		env.printSpace()
		env.writeString(name)
		if op := s.keyExist[name]; op == cmpExist || op == cmpNotExist {
			env.writeString(op2string[op])
		} else {
			env.writeString(api.SearchOperatorExist)
			env.printSpace()
			env.writeString(name)
			env.writeString(api.SearchOperatorNotExist)
		}
	}
	for _, name := range maps.Keys(s.mvals) {
		env.printExprValues(name, s.mvals[name])
	}
	if len(s.search) > 0 {
		env.printExprValues("", s.search)
	}
	env.printOrder(s.order)
	env.printPosInt(kwOffset, s.offset)
	env.printPosInt(kwLimit, s.limit)
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
		switch op := val.op; op {
		case cmpMatch:
			// An empty key signals a full-text search. Since "~" is the default op in this case,
			// it can be ignored. Therefore, print only "~" if there is a key.
			if key != "" {
				pe.writeString(api.SearchOperatorMatch)
			}
		case cmpNoMatch:
			// An empty key signals a full-text search. Since "!" is the shortcut for "!~",
			// it can be ignored. Therefore, print only "!~" if there is a key.
			if key == "" {
				pe.writeString(api.SearchOperatorNot)
			} else {
				pe.writeString(api.SearchOperatorNoMatch)
			}
		default:
			if s, found := op2string[op]; found {
				pe.writeString(s)
			} else {
				pe.writeString("|" + strconv.Itoa(int(op)))
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
	for _, name := range maps.Keys(s.keyExist) {
		if env.space {
			env.writeString(" AND ")
		}
		env.writeString(name)
		switch s.keyExist[name] {
		case cmpExist:
			env.writeString(" EXIST")
		case cmpNotExist:
			env.writeString(" NOT EXIST")
		default:
			env.writeString(" IS SCHRÖDINGER'S CAT")
		}
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
	if len(s.search) > 0 {
		if env.space {
			env.writeString(" ")
		}
		env.writeString("ANY")
		env.printHumanSelectExprValues(s.search)
		env.space = true
	}
	if s.negate {
		env.writeString(")")
		env.space = true
	}

	env.printOrder(s.order)
	env.printPosInt(kwOffset, s.offset)
	env.printPosInt(kwLimit, s.limit)
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
		switch val.op {
		case cmpHas:
			pe.writeString(" HAS ")
		case cmpHasNot:
			pe.writeString(" HAS NOT ")
		case cmpPrefix:
			pe.writeString(" PREFIX ")
		case cmpNoPrefix:
			pe.writeString(" NOT PREFIX ")
		case cmpSuffix:
			pe.writeString(" SUFFIX ")
		case cmpNoSuffix:
			pe.writeString(" NOT SUFFIX ")
		case cmpMatch:
			pe.writeString(" MATCH ")
		case cmpNoMatch:
			pe.writeString(" NOT MATCH ")
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

func (pe *printEnv) printOrder(order []sortOrder) {
	for _, o := range order {
		if o.isRandom() {
			pe.printSpace()
			pe.writeString(kwRandom)
			continue
		} else if o.key == api.KeyID && o.descending {
			continue
		}
		pe.printSpace()
		pe.writeString(kwOrder)
		if o.descending {
			pe.printSpace()
			pe.writeString(kwReverse)
		}
		pe.printSpace()
		pe.writeString(o.key)
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
