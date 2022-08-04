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
		{`a`, `ANY CONTAINS a`},
		{`:a`, `ANY CONTAINS a`},
		{`=a`, `ANY EQUAL a`},
		{`>a`, `ANY PREFIX a`},
		{`<a`, `ANY SUFFIX a`},
		{`~a`, `ANY CONTAINS a`},
		{`!a`, `ANY NOT CONTAINS a`},
		{`!:a`, `ANY NOT CONTAINS a`},
		{`!=a`, `ANY NOT EQUAL a`},
		{`!>a`, `ANY NOT PREFIX a`},
		{`!<a`, `ANY NOT SUFFIX a`},
		{`!~a`, `ANY NOT CONTAINS a`},
		{`key:`, `key MATCH ANY`},
		{`key:a`, `key MATCH a`},
	}
	for i, tc := range testcases {
		s := search.Parse(tc.spec)
		got := s.String()
		if tc.exp != got {
			t.Errorf("%d: Parse(%q) does not yield %q, but got %q", i, tc.spec, tc.exp, got)
		}
	}
}
