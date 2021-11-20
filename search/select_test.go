//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package search_test

import (
	"testing"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/search"
)

func TestMatchZidNegate(t *testing.T) {
	var s *search.Search
	s = s.AddExpr(api.KeyID, "!="+string(api.ZidVersion))
	s = s.AddExpr(api.KeyID, "!="+string(api.ZidLicense))
	matchFunc := s.CompileMatch(nil)

	testCases := []struct {
		zid api.ZettelID
		exp bool
	}{
		{api.ZidVersion, false},
		{api.ZidLicense, false},
		{api.ZidAuthors, true},
	}
	for i, tc := range testCases {
		m := meta.New(id.MustParse(tc.zid))
		if matchFunc(m) != tc.exp {
			if tc.exp {
				t.Errorf("%d: meta %v must match %v", i, m.Zid, s)
			} else {
				t.Errorf("%d: meta %v must not match %v", i, m.Zid, s)
			}
		}
	}
}
