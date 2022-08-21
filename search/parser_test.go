//-----------------------------------------------------------------------------
// Copyright (c) 2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package search_test

import (
	"testing"

	"zettelstore.de/z/search"
)

func TestParser(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		spec string
		exp  string
	}{
		{"?", "?"}, {"!?", "!?"}, {"?a", "?a"}, {"!?a", "!?a"},
		{"key?", "key?"}, {"key!?", "key!?"},
		{"b key?", "key? b"}, {"b key!?", "key!? b"},
		{"key?a", "key?a"}, {"key!?a", "key!?a"},
		{"", ""}, {"!", ""}, {":", ""}, {"!:", ""}, {">", ""}, {"!>", ""}, {"<", ""}, {"!<", ""}, {"~", ""}, {"!~", ""},
		{`a`, `a`}, {`!a`, `!a`},
		{`:a`, `:a`}, {`!:a`, `!:a`},
		{`>a`, `>a`}, {`!>a`, `!>a`},
		{`<a`, `<a`}, {`!<a`, `!<a`},
		{`~a`, `a`}, {`!~a`, `!a`},
		{`key:`, `key:`}, {`key!:`, `key!:`},
		{`key>`, `key>`}, {`key!>`, `key!>`},
		{`key<`, `key<`}, {`key!<`, `key!<`},
		{`key~`, `key~`}, {`key!~`, `key!~`},
		{`key:a`, `key:a`}, {`key!:a`, `key!:a`},
		{`key>a`, `key>a`}, {`key!>a`, `key!>a`},
		{`key<a`, `key<a`}, {`key!<a`, `key!<a`},
		{`key~a`, `key~a`}, {`key!~a`, `key!~a`},
		{`key1:a key2:b`, `key1:a key2:b`},
		{`key1: key2:b`, `key1: key2:b`},
		{"word key:a", "key:a word"},
		{`RANDOM`, `RANDOM`}, {`RANDOM a`, `a RANDOM`}, {`a RANDOM`, `a RANDOM`},
		{`RANDOM RANDOM a`, `a RANDOM`},
		{`RANDOMRANDOM a`, `RANDOMRANDOM a`}, {`a RANDOMRANDOM`, `a RANDOMRANDOM`},
		{`ORDER`, `ORDER`}, {"ORDER a b", "b ORDER a"}, {"a ORDER", "a ORDER"}, {"ORDER %", "ORDER %"},
		{"ORDER a %", "% ORDER a"},
		{"ORDER REVERSE", "ORDER REVERSE"}, {"ORDER REVERSE a b", "b ORDER REVERSE a"},
		{"a RANDOM ORDER b", "a ORDER b"}, {"a ORDER b RANDOM", "a ORDER b"},
		{"OFFSET", "OFFSET"}, {"OFFSET a", "OFFSET a"}, {"OFFSET 10 a", "a OFFSET 10"},
		{"OFFSET 01 a", "OFFSET 01 a"}, {"OFFSET 0 a", "a"}, {"a OFFSET 0", "a"},
		{"OFFSET 4 OFFSET 8", "OFFSET 8"}, {"OFFSET 8 OFFSET 4", "OFFSET 8"},
		{"LIMIT", "LIMIT"}, {"LIMIT a", "LIMIT a"}, {"LIMIT 10 a", "a LIMIT 10"},
		{"LIMIT 01 a", "LIMIT 01 a"}, {"LIMIT 0 a", "a"}, {"a LIMIT 0", "a"},
		{"LIMIT 4 LIMIT 8", "LIMIT 4"}, {"LIMIT 8 LIMIT 4", "LIMIT 4"},
		{"OR", ""}, {"OR OR", ""}, {"a OR", "a"}, {"OR b", "b"}, {"OR a OR", "a"},
		{"a OR b", "a OR b"},
	}
	for i, tc := range testcases {
		got := search.Parse(tc.spec).String()
		if tc.exp != got {
			t.Errorf("%d: Parse(%q) does not yield %q, but got %q", i, tc.spec, tc.exp, got)
			continue
		}

		gotReparse := search.Parse(got).String()
		if gotReparse != got {
			t.Errorf("%d: Parse(%q) does not yield itself, but %q", i, got, gotReparse)
		}
	}
}
