//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
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
		content := domain.NewContent(tc.s)
		got := content.IsBinary()
		if got != tc.exp {
			t.Errorf("TC=%d: expected %v, got %v", i, tc.exp, got)
		}
	}
}
