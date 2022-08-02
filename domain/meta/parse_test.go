//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package meta_test provides tests for the domain specific type 'meta'.
package meta_test

import (
	"testing"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/input"
)

func parseMetaStr(src string) *meta.Meta {
	return meta.NewFromInput(testID, input.NewInput([]byte(src)))
}

func TestEmpty(t *testing.T) {
	t.Parallel()
	m := parseMetaStr("")
	if got, ok := m.Get(api.KeySyntax); ok || got != "" {
		t.Errorf("Syntax is not %q, but %q", "", got)
	}
	if got, ok := m.GetList(api.KeyTags); ok || len(got) > 0 {
		t.Errorf("Tags are not nil, but %v", got)
	}
}

func TestTitle(t *testing.T) {
	t.Parallel()
	td := []struct{ s, e string }{
		{api.KeyTitle + ": a title", "a title"},
		{api.KeyTitle + ": a\n\t title", "a title"},
		{api.KeyTitle + ": a\n\t title\r\n  x", "a title x"},
		{api.KeyTitle + " AbC", "AbC"},
		{api.KeyTitle + " AbC\n ded", "AbC ded"},
		{api.KeyTitle + ": o\ntitle: p", "o p"},
		{api.KeyTitle + ": O\n\ntitle: P", "O"},
		{api.KeyTitle + ": b\r\ntitle: c", "b c"},
		{api.KeyTitle + ": B\r\n\r\ntitle: C", "B"},
		{api.KeyTitle + ": r\rtitle: q", "r q"},
		{api.KeyTitle + ": R\r\rtitle: Q", "R"},
	}
	for i, tc := range td {
		m := parseMetaStr(tc.s)
		if got, ok := m.Get(api.KeyTitle); !ok || got != tc.e {
			t.Log(m)
			t.Errorf("TC=%d: expected %q, got %q", i, tc.e, got)
		}
	}
}

func TestNewFromInput(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		input string
		exp   []meta.Pair
	}{
		{"", []meta.Pair{}},
		{" a:b", []meta.Pair{{"a", "b"}}},
		{"%a:b", []meta.Pair{}},
		{"a:b\r\n\r\nc:d", []meta.Pair{{"a", "b"}}},
		{"a:b\r\n%c:d", []meta.Pair{{"a", "b"}}},
		{"% a:b\r\n c:d", []meta.Pair{{"c", "d"}}},
		{"---\r\na:b\r\n", []meta.Pair{{"a", "b"}}},
		{"---\r\na:b\r\n--\r\nc:d", []meta.Pair{{"a", "b"}, {"c", "d"}}},
		{"---\r\na:b\r\n---\r\nc:d", []meta.Pair{{"a", "b"}}},
		{"---\r\na:b\r\n----\r\nc:d", []meta.Pair{{"a", "b"}}},
		{"new-title:\nnew-url:", []meta.Pair{{"new-title", ""}, {"new-url", ""}}},
	}
	for i, tc := range testcases {
		meta := parseMetaStr(tc.input)
		if got := meta.Pairs(); !equalPairs(tc.exp, got) {
			t.Errorf("TC=%d: expected=%v, got=%v", i, tc.exp, got)
		}
	}

	// Test, whether input position is correct.
	inp := input.NewInput([]byte("---\na:b\n---\nX"))
	m := meta.NewFromInput(testID, inp)
	exp := []meta.Pair{{"a", "b"}}
	if got := m.Pairs(); !equalPairs(exp, got) {
		t.Errorf("Expected=%v, got=%v", exp, got)
	}
	expCh := 'X'
	if gotCh := inp.Ch; gotCh != expCh {
		t.Errorf("Expected=%v, got=%v", expCh, gotCh)
	}
}

func equalPairs(one, two []meta.Pair) bool {
	if len(one) != len(two) {
		return false
	}
	for i := 0; i < len(one); i++ {
		if one[i].Key != two[i].Key || one[i].Value != two[i].Value {
			return false
		}
	}
	return true
}

func TestPrecursorIDSet(t *testing.T) {
	t.Parallel()
	var testdata = []struct {
		inp string
		exp string
	}{
		{"", ""},
		{"123", ""},
		{"12345678901234", "12345678901234"},
		{"123 12345678901234", "12345678901234"},
		{"12345678901234 123", "12345678901234"},
		{"01234567890123 123 12345678901234", "01234567890123 12345678901234"},
		{"12345678901234 01234567890123", "01234567890123 12345678901234"},
	}
	for i, tc := range testdata {
		m := parseMetaStr(api.KeyPrecursor + ": " + tc.inp)
		if got, ok := m.Get(api.KeyPrecursor); (!ok && tc.exp != "") || tc.exp != got {
			t.Errorf("TC=%d: expected %q, but got %q when parsing %q", i, tc.exp, got, tc.inp)
		}
	}
}
