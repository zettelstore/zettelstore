//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
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

func TestSlugify(t *testing.T) {
	tests := []struct{ in, exp string }{
		{"simple test", "simple-test"},
		{"I'm a go developer", "i-m-a-go-developer"},
		{"-!->simple   test<-!-", "simple-test"},
		{"äöüÄÖÜß", "aouaouß"},
		{"\"aèf", "aef"},
		{"a#b", "a-b"},
		{"*", ""},
	}
	for _, test := range tests {
		if got := strfun.Slugify(test.in); got != test.exp {
			t.Errorf("%q: %q != %q", test.in, got, test.exp)
		}
	}
}

func eqStringSlide(got, exp []string) bool {
	if len(got) != len(exp) {
		return false
	}
	for i, g := range got {
		if g != exp[i] {
			return false
		}
	}
	return true
}

func TestNormalizeWord(t *testing.T) {
	tests := []struct {
		in  string
		exp []string
	}{
		{"", []string{}},
		{" ", []string{}},
		{"simple test", []string{"simple", "test"}},
		{"I'm a go developer", []string{"i", "m", "a", "go", "developer"}},
		{"-!->simple   test<-!-", []string{"simple", "test"}},
		{"äöüÄÖÜß", []string{"aouaouß"}},
		{"\"aèf", []string{"aef"}},
		{"a#b", []string{"a", "b"}},
		{"*", []string{}},
		{"123", []string{"123"}},
		{"1²3", []string{"123"}},
		{"Period.", []string{"period"}},
		{" WORD  NUMBER ", []string{"word", "number"}},
	}
	for _, test := range tests {
		if got := strfun.NormalizeWords(test.in); !eqStringSlide(got, test.exp) {
			t.Errorf("%q: %q != %q", test.in, got, test.exp)
		}
	}
}
