//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package id provides domain specific types, constants, and functions about
// zettel identifier.
package id

import (
	"sort"
	"strconv"
	"time"
)

// Zid is the internal identifier of a zettel. Typically, it is a
// time stamp of the form "YYYYMMDDHHmmSS" converted to an unsigned integer.
// A zettelstore implementation should try to set the last two digits to zero,
// e.g. the seconds should be zero,
type Zid uint64

// Some important ZettelIDs
const (
	Invalid           = Zid(0) // Invalid is a Zid that will never be valid
	ConfigurationZid  = Zid(100)
	BaseTemplateZid   = Zid(10100)
	LoginTemplateZid  = Zid(10200)
	ListTemplateZid   = Zid(10300)
	DetailTemplateZid = Zid(10401)
	InfoTemplateZid   = Zid(10402)
	FormTemplateZid   = Zid(10403)
	RenameTemplateZid = Zid(10404)
	DeleteTemplateZid = Zid(10405)
	RolesTemplateZid  = Zid(10500)
	TagsTemplateZid   = Zid(10600)
	BaseCSSZid        = Zid(20001)

	// Range 90000...99999 is reserved for zettel templates
	TemplateNewZettelZid = Zid(91001)
	TemplateNewUserZid   = Zid(96001)

	WelcomeZid = Zid(19700101000000)
)

const maxZid = 99999999999999

// Parse interprets a string as a zettel identification and
// returns its integer value.
func Parse(s string) (Zid, error) {
	if len(s) != 14 {
		return Invalid, strconv.ErrSyntax
	}
	res, err := strconv.ParseUint(s, 10, 47)
	if err != nil {
		return Invalid, err
	}
	if res == 0 {
		return Invalid, strconv.ErrRange
	}
	return Zid(res), nil
}

const digits = "0123456789"

// String converts the zettel identification to a string of 14 digits.
// Only defined for valid ids.
func (zid Zid) String() string {
	return string(zid.Bytes())
}

// Bytes converts the zettel identification to a byte slice of 14 digits.
// Only defined for valid ids.
func (zid Zid) Bytes() []byte {
	result := make([]byte, 14)
	for i := 13; i >= 0; i-- {
		result[i] = digits[zid%10]
		zid /= 10
	}
	return result
}

// IsValid determines if zettel id is a valid one, e.g. consists of max. 14 digits.
func (zid Zid) IsValid() bool { return 0 < zid && zid <= maxZid }

// New returns a new zettel id based on the current time.
func New(withSeconds bool) Zid {
	now := time.Now()
	var s string
	if withSeconds {
		s = now.Format("20060102150405")
	} else {
		s = now.Format("20060102150400")
	}
	res, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return res
}

// Sort a slice of Zids.
func Sort(zids []Zid) {
	sort.Sort(zidSlice(zids))
}

type zidSlice []Zid

func (zs zidSlice) Len() int           { return len(zs) }
func (zs zidSlice) Less(i, j int) bool { return zs[i] < zs[j] }
func (zs zidSlice) Swap(i, j int)      { zs[i], zs[j] = zs[j], zs[i] }
