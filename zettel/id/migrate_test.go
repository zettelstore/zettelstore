//-----------------------------------------------------------------------------
// Copyright (c) 2024-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2024-present Detlef Stern
//-----------------------------------------------------------------------------

package id_test

import (
	"testing"

	"zettelstore.de/z/zettel/id"
)

func TestMigrat(t *testing.T) {
	testcases := []struct {
		inp []id.Zid
		err string
		exp []id.ZidN
	}{
		{[]id.Zid{1, 2, 3}, "", []id.ZidN{1, 2, 3}},
		{[]id.Zid{3, 2, 1}, "", []id.ZidN{3, 2, 1}},
		{[]id.Zid{20240224123456, 19700101000000}, "out of sequence: 19700101000000", []id.ZidN{46657, 1}},
	}
	for i, tc := range testcases {
		migrator := id.NewZidMigrator()
		for pos, zidO := range tc.inp {
			zid, err := migrator.Migrate(zidO)
			if err != nil {
				if tc.err == "" {
					t.Errorf("%d: no error expected, but got: %v", i, err)
				} else if err.Error() != tc.err {
					t.Errorf("%d: error %v expected, but got %v", i, tc.err, err)
				}
				continue
			}
			if exp := tc.exp[pos]; exp != zid {
				t.Errorf("%d: %v should migrate to %v, but got: %v", i, zidO, exp, zid)
			}
		}
	}
}
