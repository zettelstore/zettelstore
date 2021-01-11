//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package strfun provides some string functions.
package strfun_test

import (
	"testing"

	"zettelstore.de/z/strfun"
)

var tests = []struct{ in, exp string }{
	{"simple test", "simple-test"},
	{"I'm a go developer", "i-m-a-go-developer"},
	{"-!->simple   test<-!-", "simple-test"},
	{"äöüÄÖÜß", "aouaouß"},
	{"\"aèf", "aef"},
	{"a#b", "a-b"},
	{"*", ""},
}

func TestSlugify(t *testing.T) {
	for _, test := range tests {
		if got := strfun.Slugify(test.in); got != test.exp {
			t.Errorf("%q: %q != %q", test.in, got, test.exp)
		}
	}
}
