//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package place provides a generic interface to zettel places.
package place

import (
	"time"

	"zettelstore.de/z/domain/id"
)

func GetNewZid(testZid func(id.Zid) (bool, error)) (id.Zid, error) {
	zid := id.New(false)
	found, err := testZid(zid)
	if err != nil {
		return id.Invalid, err
	}
	if found {
		return zid, nil
	}
	for {
		zid = id.New(true)
		found, err := testZid(zid)
		if err != nil {
			return id.Invalid, err
		}
		if found {
			return zid, nil
		}
		// TODO: do not wait here, but in a non-blocking goroutine.
		time.Sleep(100 * time.Millisecond)
	}
}
