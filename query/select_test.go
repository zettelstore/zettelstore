//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2021-present Detlef Stern
//-----------------------------------------------------------------------------

package query_test

import (
	"context"
	"testing"

	"t73f.de/r/zsc/api"
	"zettelstore.de/z/query"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

func TestMatchZidNegate(t *testing.T) {
	q := query.Parse(api.KeyID + api.SearchOperatorHasNot + string(api.ZidVersion) + " " + api.KeyID + api.SearchOperatorHasNot + string(api.ZidLicense))
	compiled := q.RetrieveAndCompile(context.Background(), nil, nil)

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
				t.Errorf("%d: meta %v must match %q", i, m.Zid, q)
			} else {
				t.Errorf("%d: meta %v must not match %q", i, m.Zid, q)
			}
		}
	}
}
