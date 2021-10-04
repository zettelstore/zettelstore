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
	"strconv"
	"time"

	"zettelstore.de/c/api"
)

// Zid is the internal identifier of a zettel. Typically, it is a
// time stamp of the form "YYYYMMDDHHmmSS" converted to an unsigned integer.
// A zettelstore implementation should try to set the last two digits to zero,
// e.g. the seconds should be zero,
type Zid uint64

// Some important ZettelIDs.
const (
	Invalid = Zid(0) // Invalid is a Zid that will never be valid
)

// ZettelIDs that are used as Zid more than once.
// Note: if you change some values, ensure that you also change them in the
//       constant box. They are mentioned there literally, because these
//       constants are not available there.
var (
	ConfigurationZid   = MustParse(api.ZidConfiguration)
	BaseTemplateZid    = MustParse(api.ZidBaseTemplate)
	LoginTemplateZid   = MustParse(api.ZidLoginTemplate)
	ListTemplateZid    = MustParse(api.ZidListTemplate)
	ZettelTemplateZid  = MustParse(api.ZidZettelTemplate)
	InfoTemplateZid    = MustParse(api.ZidInfoTemplate)
	FormTemplateZid    = MustParse(api.ZidFormTemplate)
	RenameTemplateZid  = MustParse(api.ZidRenameTemplate)
	DeleteTemplateZid  = MustParse(api.ZidDeleteTemplate)
	ContextTemplateZid = MustParse(api.ZidContextTemplate)
	RolesTemplateZid   = MustParse(api.ZidRolesTemplate)
	TagsTemplateZid    = MustParse(api.ZidTagsTemplate)
	ErrorTemplateZid   = MustParse(api.ZidErrorTemplate)
	EmojiZid           = MustParse(api.ZidEmoji)
	TOCNewTemplateZid  = MustParse(api.ZidTOCNewTemplate)
	DefaultHomeZid     = MustParse(api.ZidDefaultHome)
)

const maxZid = 99999999999999

// ParseUint interprets a string as a possible zettel identifier
// and returns its integer value.
func ParseUint(s string) (uint64, error) {
	res, err := strconv.ParseUint(s, 10, 47)
	if err != nil {
		return 0, err
	}
	if res == 0 || res > maxZid {
		return res, strconv.ErrRange
	}
	return res, nil
}

// Parse interprets a string as a zettel identification and
// returns its value.
func Parse(s string) (Zid, error) {
	if len(s) != 14 {
		return Invalid, strconv.ErrSyntax
	}
	res, err := ParseUint(s)
	if err != nil {
		return Invalid, err
	}
	return Zid(res), nil
}

// MustParse tries to interpret a string as a zettel identifier and returns
// its value or panics otherwise.
func MustParse(s api.ZettelID) Zid {
	zid, err := Parse(string(s))
	if err == nil {
		return zid
	}
	panic(err)
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
	n := uint64(zid)
	result := make([]byte, 14)
	for i := 13; i >= 0; i-- {
		result[i] = digits[n%10]
		n /= 10
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
