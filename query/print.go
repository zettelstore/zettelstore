//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package query

import (
	"io"
	"strconv"
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/c/maps"
)

var op2string = map[compareOp]string{
	cmpExist:    api.ExistOperator,
	cmpNotExist: api.ExistNotOperator,
	cmpHas:      api.SearchOperatorHas,
	cmpHasNot:   api.SearchOperatorHasNot,
	cmpPrefix:   api.SearchOperatorPrefix,
	cmpNoPrefix: api.SearchOperatorNoPrefix,
	cmpSuffix:   api.SearchOperatorSuffix,
	cmpNoSuffix: api.SearchOperatorNoSuffix,
	cmpMatch:    api.SearchOperatorMatch,
	cmpNoMatch:  api.SearchOperatorNoMatch,
}

func (q *Query) String() string {
	var sb strings.Builder
	q.Print(&sb)
	return sb.String()
}

// Print the query in a parseable form.
func (q *Query) Print(w io.Writer) {
	if q == nil {
		return
	}
	env := printEnv{w: w}
	env.printContext(q)
	for i, term := range q.terms {
		if i > 0 {
			env.writeString(" OR")
		}
		for _, name := range maps.Keys(term.keys) {
			env.printSpace()
			env.writeString(name)
			if op := term.keys[name]; op == cmpExist || op == cmpNotExist {
				env.writeString(op2string[op])
			} else {
				env.writeString(api.ExistOperator)
				env.printSpace()
				env.writeString(name)
				env.writeString(api.ExistNotOperator)
			}
		}
		for _, name := range maps.Keys(term.mvals) {
			env.printExprValues(name, term.mvals[name])
		}
		if len(term.search) > 0 {
			env.printExprValues("", term.search)
		}
	}
	env.printPosInt(kwPick, q.pick)
	env.printOrder(q.order)
	env.printPosInt(kwOffset, q.offset)
	env.printPosInt(kwLimit, q.limit)
	env.printActions(q.actions)
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
func (pe *printEnv) write(ch byte)        { pe.w.Write([]byte{ch}) }
func (pe *printEnv) writeString(s string) { io.WriteString(pe.w, s) }

func (pe *printEnv) printContext(q *Query) {
	if zid := q.zid; zid.IsValid() {
		pe.writeString(kwContext)
		pe.space = true
		pe.printSpace()
		pe.writeString(zid.String())
		switch q.dir {
		case contextBackward:
			pe.printSpace()
			pe.writeString(kwBackward)
		case contextForward:
			pe.printSpace()
			pe.writeString(kwForward)
		}
		pe.printPosInt(kwCost, q.maxCost)
		pe.printPosInt(kwMax, q.maxCount)
		// pe.writeString("!")
	}

}
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
				pe.writeString("%" + strconv.Itoa(int(op)))
			}
		}
		if s := val.value; s != "" {
			pe.writeString(s)
		}
	}
}

func (q *Query) Human() string {
	var sb strings.Builder
	q.PrintHuman(&sb)
	return sb.String()
}

// PrintHuman the query to a writer in a human readable form.
func (q *Query) PrintHuman(w io.Writer) {
	if q == nil {
		return
	}
	env := printEnv{w: w}
	env.printContext(q)
	for i, term := range q.terms {
		if i > 0 {
			env.writeString(" OR ")
			env.space = false
		}
		for _, name := range maps.Keys(term.keys) {
			if env.space {
				env.writeString(" AND ")
			}
			env.writeString(name)
			switch term.keys[name] {
			case cmpExist:
				env.writeString(" EXIST")
			case cmpNotExist:
				env.writeString(" NOT EXIST")
			default:
				env.writeString(" IS SCHRÖDINGER'S CAT")
			}
			env.space = true
		}
		for _, name := range maps.Keys(term.mvals) {
			if env.space {
				env.writeString(" AND ")
			}
			env.writeString(name)
			env.printHumanSelectExprValues(term.mvals[name])
			env.space = true
		}
		if len(term.search) > 0 {
			if env.space {
				env.writeString(" ")
			}
			env.writeString("ANY")
			env.printHumanSelectExprValues(term.search)
			env.space = true
		}
	}

	env.printPosInt(kwPick, q.pick)
	env.printOrder(q.order)
	env.printPosInt(kwOffset, q.offset)
	env.printPosInt(kwLimit, q.limit)
	env.printActions(q.actions)
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
			pe.writeString("NOTHING")
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

func (pe *printEnv) printActions(words []string) {
	if len(words) > 0 {
		pe.printSpace()
		pe.write(actionSeparatorChar)
		for _, word := range words {
			pe.printSpace()
			pe.writeString(word)
		}
	}
}
