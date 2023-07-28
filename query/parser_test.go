//-----------------------------------------------------------------------------
// Copyright (c) 2022-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package query_test

import (
	"testing"

	"zettelstore.de/z/query"
)

func TestParser(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		spec string
		exp  string
	}{
		{"1", "1"}, // Just a number will transform to search for that numer in all zettel

		{"1 IDENT", "00000000000001 IDENT"},
		{"IDENT", "IDENT"},

		{"1 ITEMS", "00000000000001 ITEMS"},
		{"ITEMS", "ITEMS"},

		{"CONTEXT", "CONTEXT"}, {"CONTEXT a", "CONTEXT a"},
		{"0 CONTEXT", "0 CONTEXT"}, {"1 CONTEXT", "00000000000001 CONTEXT"},
		{"00000000000001 CONTEXT", "00000000000001 CONTEXT"},
		{"100000000000001 CONTEXT", "100000000000001 CONTEXT"},
		{"1 CONTEXT BACKWARD", "00000000000001 CONTEXT BACKWARD"},
		{"1 CONTEXT FORWARD", "00000000000001 CONTEXT FORWARD"},
		{"1 CONTEXT COST ", "00000000000001 CONTEXT COST"},
		{"1 CONTEXT COST 3", "00000000000001 CONTEXT COST 3"}, {"1 CONTEXT COST x", "00000000000001 CONTEXT COST x"},
		{"1 CONTEXT MAX 5", "00000000000001 CONTEXT MAX 5"}, {"1 CONTEXT MAX y", "00000000000001 CONTEXT MAX y"},
		{"1 CONTEXT MAX 5 COST 7", "00000000000001 CONTEXT COST 7 MAX 5"},
		{"1 CONTEXT |  N", "00000000000001 CONTEXT | N"},
		{"1 1 CONTEXT", "00000000000001 CONTEXT"},
		{"1 2 CONTEXT", "00000000000001 00000000000002 CONTEXT"},
		{"2 1 CONTEXT", "00000000000002 00000000000001 CONTEXT"},

		{"CONTEXT 0", "CONTEXT 0"}, {"CONTEXT 1", "00000000000001 CONTEXT"},
		{"CONTEXT 00000000000001", "00000000000001 CONTEXT"},
		{"CONTEXT 100000000000001", "CONTEXT 100000000000001"},
		{"CONTEXT 1 BACKWARD", "00000000000001 CONTEXT BACKWARD"},
		{"CONTEXT 1 FORWARD", "00000000000001 CONTEXT FORWARD"},
		{"CONTEXT 1 COST 3", "00000000000001 CONTEXT COST 3"}, {"CONTEXT 1 COST x", "00000000000001 CONTEXT COST x"},
		{"CONTEXT 1 MAX 5", "00000000000001 CONTEXT MAX 5"}, {"CONTEXT 1 MAX y", "00000000000001 CONTEXT MAX y"},
		{"CONTEXT 1 MAX 5 COST 7", "00000000000001 CONTEXT COST 7 MAX 5"},
		{"CONTEXT 1 |  N", "00000000000001 CONTEXT | N"},

		{"?", "?"}, {"!?", "!?"}, {"?a", "?a"}, {"!?a", "!?a"},
		{"key?", "key?"}, {"key!?", "key!?"},
		{"b key?", "key? b"}, {"b key!?", "key!? b"},
		{"key?a", "key?a"}, {"key!?a", "key!?a"},
		{"", ""}, {"!", ""}, {":", ""}, {"!:", ""}, {"[", ""}, {"![", ""}, {"]", ""}, {"!]", ""}, {"~", ""}, {"!~", ""}, {"<", ""}, {"!<", ""}, {">", ""}, {"!>", ""},
		{`a`, `a`}, {`!a`, `!a`},
		{`=a`, `=a`}, {`!=a`, `!=a`},
		{`:a`, `:a`}, {`!:a`, `!:a`},
		{`[a`, `[a`}, {`![a`, `![a`},
		{`]a`, `]a`}, {`!]a`, `!]a`},
		{`~a`, `a`}, {`!~a`, `!a`},
		{`key=`, `key=`}, {`key!=`, `key!=`},
		{`key:`, `key:`}, {`key!:`, `key!:`},
		{`key[`, `key[`}, {`key![`, `key![`},
		{`key]`, `key]`}, {`key!]`, `key!]`},
		{`key~`, `key~`}, {`key!~`, `key!~`},
		{`key<`, `key<`}, {`key!<`, `key!<`},
		{`key>`, `key>`}, {`key!>`, `key!>`},
		{`key=a`, `key=a`}, {`key!=a`, `key!=a`},
		{`key:a`, `key:a`}, {`key!:a`, `key!:a`},
		{`key[a`, `key[a`}, {`key![a`, `key![a`},
		{`key]a`, `key]a`}, {`key!]a`, `key!]a`},
		{`key~a`, `key~a`}, {`key!~a`, `key!~a`},
		{`key<a`, `key<a`}, {`key!<a`, `key!<a`},
		{`key>a`, `key>a`}, {`key!>a`, `key!>a`},
		{`key1:a key2:b`, `key1:a key2:b`},
		{`key1: key2:b`, `key1: key2:b`},
		{"word key:a", "key:a word"},
		{`PICK 3`, `PICK 3`}, {`PICK 9 PICK 11`, `PICK 9`},
		{"PICK a", "PICK a"},
		{`RANDOM`, `RANDOM`}, {`RANDOM a`, `a RANDOM`}, {`a RANDOM`, `a RANDOM`},
		{`RANDOM RANDOM a`, `a RANDOM`},
		{`RANDOMRANDOM a`, `RANDOMRANDOM a`}, {`a RANDOMRANDOM`, `a RANDOMRANDOM`},
		{`ORDER`, `ORDER`}, {"ORDER a b", "b ORDER a"}, {"a ORDER", "a ORDER"}, {"ORDER %", "ORDER %"},
		{"ORDER a %", "% ORDER a"},
		{"ORDER REVERSE", "ORDER REVERSE"}, {"ORDER REVERSE a b", "b ORDER REVERSE a"},
		{"a RANDOM ORDER b", "a ORDER b"}, {"a ORDER b RANDOM", "a ORDER b"},
		{"OFFSET", "OFFSET"}, {"OFFSET a", "OFFSET a"}, {"OFFSET 10 a", "a OFFSET 10"},
		{"OFFSET 01 a", "a OFFSET 1"}, {"OFFSET 0 a", "a"}, {"a OFFSET 0", "a"},
		{"OFFSET 4 OFFSET 8", "OFFSET 8"}, {"OFFSET 8 OFFSET 4", "OFFSET 8"},
		{"LIMIT", "LIMIT"}, {"LIMIT a", "LIMIT a"}, {"LIMIT 10 a", "a LIMIT 10"},
		{"LIMIT 01 a", "a LIMIT 1"}, {"LIMIT 0 a", "a"}, {"a LIMIT 0", "a"},
		{"LIMIT 4 LIMIT 8", "LIMIT 4"}, {"LIMIT 8 LIMIT 4", "LIMIT 4"},
		{"OR", ""}, {"OR OR", ""}, {"a OR", "a"}, {"OR b", "b"}, {"OR a OR", "a"},
		{"a OR b", "a OR b"},
		{"|", ""}, {" | RANDOM", "| RANDOM"}, {"| RANDOM", "| RANDOM"}, {"a|a b ", "a | a b"},
	}
	for i, tc := range testcases {
		got := query.Parse(tc.spec).String()
		if tc.exp != got {
			t.Errorf("%d: Parse(%q) does not yield %q, but got %q", i, tc.spec, tc.exp, got)
			continue
		}

		gotReparse := query.Parse(got).String()
		if gotReparse != got {
			t.Errorf("%d: Parse(%q) does not yield itself, but %q", i, got, gotReparse)
		}

		gotPipe := query.Parse(got + "|").String()
		if gotPipe != got {
			t.Errorf("%d: Parse(%q) does not yield itself, but %q", i, got+"|", gotReparse)
		}
	}
}
