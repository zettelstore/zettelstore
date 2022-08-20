//-----------------------------------------------------------------------------
// Copyright (c) 2021-2022 Detlef Stern
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

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/search"
)

func TestMatchZidNegate(t *testing.T) {
	s := search.Parse(api.KeyID + api.SearchOperatorHasNot + string(api.ZidVersion) + " " + api.KeyID + api.SearchOperatorHasNot + string(api.ZidLicense))
	compiled := s.RetrieveAndCompile(nil)

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
		if compiled.Terms[0].Match(m) != tc.exp {
			if tc.exp {
				t.Errorf("%d: meta %v must match %q", i, m.Zid, s)
			} else {
				t.Errorf("%d: meta %v must not match %q", i, m.Zid, s)
			}
		}
	}
}
