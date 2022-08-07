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
	testcases := []struct {
		spec string
		exp  string
	}{
		{"", ""},
		{`a`, `a`}, {`!a`, `!a`},
		{`:a`, `a`}, {`!:a`, `!a`},
		{`=a`, `=a`}, {`!=a`, `!=a`},
		{`>a`, `>a`}, {`!>a`, `!>a`},
		{`<a`, `<a`}, {`!<a`, `!<a`},
		{`~a`, `a`}, {`!~a`, `!a`},
		{`key:`, `key:`}, {`key!:`, `key!:`},
		{`key=`, `key=`}, {`key!=`, `key!=`},
		{`key>`, `key>`}, {`key!>`, `key!>`},
		{`key<`, `key<`}, {`key!<`, `key!<`},
		{`key~`, `key~`}, {`key!~`, `key!~`},
		{`key:a`, `key:a`}, {`key!:a`, `key!:a`},
		{`key=a`, `key=a`}, {`key!=a`, `key!=a`},
		{`key>a`, `key>a`}, {`key!>a`, `key!>a`},
		{`key<a`, `key<a`}, {`key!<a`, `key!<a`},
		{`key~a`, `key~a`}, {`key!~a`, `key!~a`},
		{`key1:a key2:b`, `key1:a key2:b`},
		{`key1: key2:b`, `key1: key2:b`},
		{`NEGATE`, `NEGATE`},
		{`NEGATE a`, `NEGATE a`},
		{`a NEGATE`, `NEGATE a`},
		{`NEGATE NEGATE a`, `a`},
		{`NEGATENEGATE a`, `NEGATENEGATE a`},
	}
	for i, tc := range testcases {
		s := search.Parse(tc.spec)
		got := s.String()
		if tc.exp != got {
			t.Errorf("%d: Parse(%q) does not yield %q, but got %q", i, tc.spec, tc.exp, got)
			continue
		}
		got2 := search.Parse(got).String()
		if got2 != got {
			t.Errorf("%d: Parse(%q) does not yield itself, but %q", i, got, got2)
		}
	}
}