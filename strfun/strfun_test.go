//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
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

func TestTrimSpaceRight(t *testing.T) {
	const space = "\t\v\r\f\n\u0085\u00a0\u2000\u3000"
	testcases := []struct {
		in  string
		exp string
	}{
		{"", ""},
		{"abc", "abc"},
		{" ", ""},
		{space, ""},
		{space + "abc" + space, space + "abc"},
		{" \t\r\n \t\t\r\r\n\n ", ""},
		{" \t\r\n x\t\t\r\r\n\n ", " \t\r\n x"},
		{" \u2000\t\r\n x\t\t\r\r\ny\n \u3000", " \u2000\t\r\n x\t\t\r\r\ny"},
		{"1 \t\r\n2", "1 \t\r\n2"},
		{" x\x80", " x\x80"},
		{" x\xc0", " x\xc0"},
		{"x \xc0\xc0 ", "x \xc0\xc0"},
		{"x \xc0", "x \xc0"},
		{"x \xc0 ", "x \xc0"},
		{"x \xc0\xc0 ", "x \xc0\xc0"},
		{"x ☺\xc0\xc0 ", "x ☺\xc0\xc0"},
		{"x ☺ ", "x ☺"},
	}
	for i, tc := range testcases {
		got := strfun.TrimSpaceRight(tc.in)
		if got != tc.exp {
			t.Errorf("%d/%q: expected %q, got %q", i, tc.in, tc.exp, got)
		}
	}
}

func TestLength(t *testing.T) {
	testcases := []struct {
		in  string
		exp int
	}{
		{"", 0},
		{"äbc", 3},
	}
	for i, tc := range testcases {
		got := strfun.Length(tc.in)
		if got != tc.exp {
			t.Errorf("%d/%q: expected %v, got %v", i, tc.in, tc.exp, got)
		}
	}
}

func TestJustifyLeft(t *testing.T) {
	testcases := []struct {
		in  string
		ml  int
		exp string
	}{
		{"", 0, ""},
		{"äbc", 0, ""},
		{"äbc", 1, "\u2025"},
		{"äbc", 2, "ä\u2025"},
		{"äbc", 3, "äbc"},
		{"äbc", 4, "äbc:"},
	}
	for i, tc := range testcases {
		got := strfun.JustifyLeft(tc.in, tc.ml, ':')
		if got != tc.exp {
			t.Errorf("%d/%q/%d: expected %q, got %q", i, tc.in, tc.ml, tc.exp, got)
		}
	}
}
