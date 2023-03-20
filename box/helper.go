//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package box

import (
	"net/url"
	"strconv"
	"time"

	"zettelstore.de/z/domain/id"
)

// GetNewZid calculates a new and unused zettel identifier, based on the current date and time.
func GetNewZid(testZid func(id.Zid) (bool, error)) (id.Zid, error) {
	withSeconds := false
	for i := 0; i < 90; i++ { // Must be completed within 9 seconds (less than web/server.writeTimeout)
		zid := id.New(withSeconds)
		found, err := testZid(zid)
		if err != nil {
			return id.Invalid, err
		}
		if found {
			return zid, nil
		}
		// TODO: do not wait here unconditionally.
		time.Sleep(100 * time.Millisecond)
		withSeconds = true
	}
	return id.Invalid, ErrConflict
}

// GetQueryBool is a helper function to extract bool values from a box URI.
func GetQueryBool(u *url.URL, key string) bool {
	_, ok := u.Query()[key]
	return ok
}

// GetQueryInt is a helper function to extract int values of a specified range from a box URI.
func GetQueryInt(u *url.URL, key string, min, def, max int) int {
	sVal := u.Query().Get(key)
	if sVal == "" {
		return def
	}
	iVal, err := strconv.Atoi(sVal)
	if err != nil {
		return def
	}
	if iVal < min {
		return min
	}
	if iVal > max {
		return max
	}
	return iVal
}
