//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package meta provides the domain specific type 'meta'.
package meta

import (
	"sort"
	"strings"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/input"
	"zettelstore.de/z/runes"
)

// NewFromInput parses the meta data of a zettel.
func NewFromInput(zid id.Zid, inp *input.Input) *Meta {
	if inp.Ch == '-' && inp.PeekN(0) == '-' && inp.PeekN(1) == '-' {
		skipToEOL(inp)
		inp.EatEOL()
	}
	meta := New(zid)
	for {
		skipSpace(inp)
		switch inp.Ch {
		case '\r':
			if inp.Peek() == '\n' {
				inp.Next()
			}
			fallthrough
		case '\n':
			inp.Next()
			return meta
		case input.EOS:
			return meta
		case '%':
			skipToEOL(inp)
			inp.EatEOL()
			continue
		}
		parseHeader(meta, inp)
		if inp.Ch == '-' && inp.PeekN(0) == '-' && inp.PeekN(1) == '-' {
			skipToEOL(inp)
			inp.EatEOL()
			meta.YamlSep = true
			return meta
		}
	}
}

func parseHeader(m *Meta, inp *input.Input) {
	pos := inp.Pos
	for isHeader(inp.Ch) {
		inp.Next()
	}
	key := inp.Src[pos:inp.Pos]
	skipSpace(inp)
	if inp.Ch == ':' {
		inp.Next()
	}
	var val string
	for {
		skipSpace(inp)
		pos = inp.Pos
		skipToEOL(inp)
		val += inp.Src[pos:inp.Pos]
		inp.EatEOL()
		if !runes.IsSpace(inp.Ch) {
			break
		}
		val += " "
	}
	addToMeta(m, key, val)
}

func skipSpace(inp *input.Input) {
	for runes.IsSpace(inp.Ch) {
		inp.Next()
	}
}

func skipToEOL(inp *input.Input) {
	for {
		switch inp.Ch {
		case '\n', '\r', input.EOS:
			return
		}
		inp.Next()
	}
}

// Return true iff rune is valid for header key.
func isHeader(ch rune) bool {
	return ('a' <= ch && ch <= 'z') ||
		('0' <= ch && ch <= '9') ||
		ch == '-' ||
		('A' <= ch && ch <= 'Z')
}

type predValidElem func(string) bool

func addToSet(set map[string]bool, elems []string, useElem predValidElem) {
	for _, s := range elems {
		if len(s) > 0 && useElem(s) {
			set[s] = true
		}
	}
}

func addSet(m *Meta, key, val string, useElem predValidElem) {
	newElems := strings.Fields(val)
	oldElems, ok := m.GetList(key)
	if !ok {
		oldElems = nil
	}

	set := make(map[string]bool, len(newElems)+len(oldElems))
	addToSet(set, newElems, useElem)
	if len(set) == 0 {
		// Nothing to add. Maybe because of filtered elements.
		return
	}
	addToSet(set, oldElems, useElem)

	resultList := make([]string, 0, len(set))
	for tag := range set {
		resultList = append(resultList, tag)
	}
	sort.Strings(resultList)
	m.SetList(key, resultList)
}

func addData(m *Meta, k, v string) {
	if o, ok := m.Get(k); !ok || o == "" {
		m.Set(k, v)
	} else if v != "" {
		m.Set(k, o+" "+v)
	}
}

func addToMeta(m *Meta, key, val string) {
	v := trimValue(val)
	key = strings.ToLower(key)
	if !KeyIsValid(key) {
		return
	}
	switch key {
	case "", KeyID:
		// Empty key and 'id' key will be ignored
		return
	}

	switch KeyType(key) {
	case TypeString, TypeZettelmarkup:
		if v != "" {
			addData(m, key, v)
		}
	case TypeTagSet:
		addSet(m, key, v, func(s string) bool { return s[0] == '#' })
	case TypeWord:
		m.Set(key, strings.ToLower(v))
	case TypeWordSet:
		addSet(m, key, strings.ToLower(v), func(s string) bool { return true })
	case TypeID:
		if _, err := id.Parse(v); err == nil {
			m.Set(key, v)
		}
	case TypeIDSet:
		addSet(m, key, v, func(s string) bool {
			_, err := id.Parse(s)
			return err == nil
		})
	case TypeTimestamp:
		if _, ok := TimeValue(v); ok {
			m.Set(key, v)
		}
	case TypeEmpty:
		fallthrough
	default:
		addData(m, key, v)
	}
}
