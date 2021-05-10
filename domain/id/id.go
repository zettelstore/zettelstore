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
)

// Zid is the internal identifier of a zettel. Typically, it is a
// time stamp of the form "YYYYMMDDHHmmSS" converted to an unsigned integer.
// A zettelstore implementation should try to set the last two digits to zero,
// e.g. the seconds should be zero,
type Zid uint64

// Some important ZettelIDs.
// Note: if you change some values, ensure that you also change them in the
//       constant place. They are mentioned there literally, because these
//       constants are not available there.
const (
	Invalid = Zid(0) // Invalid is a Zid that will never be valid

	// System zettel
	VersionZid              = Zid(1)
	HostZid                 = Zid(2)
	OperatingSystemZid      = Zid(3)
	LicenseZid              = Zid(4)
	AuthorsZid              = Zid(5)
	DependenciesZid         = Zid(6)
	EnvironmentZid          = Zid(10)
	MetricsZid              = Zid(12)
	IndexerZid              = Zid(18)
	PlaceManagerZid         = Zid(20)
	MetadataKeyZid          = Zid(90)
	StartupConfigurationZid = Zid(96)
	ConfigurationZid        = Zid(100)

	// WebUI HTML templates are in the range 10000..19999
	BaseTemplateZid    = Zid(10100)
	LoginTemplateZid   = Zid(10200)
	ListTemplateZid    = Zid(10300)
	ZettelTemplateZid  = Zid(10401)
	InfoTemplateZid    = Zid(10402)
	FormTemplateZid    = Zid(10403)
	RenameTemplateZid  = Zid(10404)
	DeleteTemplateZid  = Zid(10405)
	ContextTemplateZid = Zid(10406)
	RolesTemplateZid   = Zid(10500)
	TagsTemplateZid    = Zid(10600)
	ErrorTemplateZid   = Zid(10700)

	// WebUI CSS zettel are in the range 20000..29999
	BaseCSSZid = Zid(20001)

	// WebUI JS zettel are in the range 30000..39999

	// WebUI image zettel are in the range 40000..49999
	EmojiZid = Zid(40001)

	// Range 90000...99999 is reserved for zettel templates
	TOCNewTemplateZid    = Zid(90000)
	TemplateNewZettelZid = Zid(90001)
	TemplateNewUserZid   = Zid(90002)

	DefaultHomeZid = Zid(10000000000)
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
