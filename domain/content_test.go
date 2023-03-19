//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package domain_test

import (
	"testing"

	"zettelstore.de/z/domain"
)

func TestContentIsBinary(t *testing.T) {
	t.Parallel()
	td := []struct {
		s   string
		exp bool
	}{
		{"abc", false},
		{"äöü", false},
		{"", false},
		{string([]byte{0}), true},
	}
	for i, tc := range td {
		content := domain.NewContent([]byte(tc.s))
		got := content.IsBinary()
		if got != tc.exp {
			t.Errorf("TC=%d: expected %v, got %v", i, tc.exp, got)
		}
	}
}

func TestTrimSpace(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		in, exp string
	}{
		{"", ""},
		{" ", ""},
		{"abc", "abc"},
		{" abc", " abc"},
		{"abc ", "abc"},
		{"abc \n", "abc"},
		{"abc\n ", "abc"},
		{"\nabc", "abc"},
		{" \nabc", "abc"},
		{" \n abc", " abc"},
		{" \n\n abc", " abc"},
		{" \n \n abc", " abc"},
		{" \n \n abc \n \n ", " abc"},
	}
	for _, tc := range testcases {
		c := domain.NewContent([]byte(tc.in))
		c.TrimSpace()
		got := c.AsString()
		if got != tc.exp {
			t.Errorf("TrimSpace(%q) should be %q, but got %q", tc.in, tc.exp, got)
		}
	}
}
