//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2020-present Detlef Stern
//-----------------------------------------------------------------------------

// Package id provides zettel specific types, constants, and functions about
// zettel identifier.
package id

import (
	"strconv"
	"time"

	"t73f.de/r/zsc/api"
)

// ZidO is the internal identifier of a zettel. Typically, it is a
// time stamp of the form "YYYYMMDDHHmmSS" converted to an unsigned integer.
// A zettelstore implementation should try to set the last two digits to zero,
// e.g. the seconds should be zero,
type ZidO uint64

// Some important ZettelIDs.
const (
	InvalidO = ZidO(0) // Invalid is a Zid that will never be valid
)

// ZettelIDs that are used as Zid more than once.
//
// Note: if you change some values, ensure that you also change them in the
// Constant box. They are mentioned there literally, because these
// constants are not available there.
var (
	ConfigurationZidO  = MustParseO(api.ZidConfiguration)
	BaseTemplateZidO   = MustParseO(api.ZidBaseTemplate)
	LoginTemplateZidO  = MustParseO(api.ZidLoginTemplate)
	ListTemplateZidO   = MustParseO(api.ZidListTemplate)
	ZettelTemplateZidO = MustParseO(api.ZidZettelTemplate)
	InfoTemplateZidO   = MustParseO(api.ZidInfoTemplate)
	FormTemplateZidO   = MustParseO(api.ZidFormTemplate)
	RenameTemplateZidO = MustParseO(api.ZidRenameTemplate)
	DeleteTemplateZidO = MustParseO(api.ZidDeleteTemplate)
	ErrorTemplateZidO  = MustParseO(api.ZidErrorTemplate)
	StartSxnZidO       = MustParseO(api.ZidSxnStart)
	BaseSxnZidO        = MustParseO(api.ZidSxnBase)
	PreludeSxnZidO     = MustParseO(api.ZidSxnPrelude)
	EmojiZidO          = MustParseO(api.ZidEmoji)
	TOCNewTemplateZidO = MustParseO(api.ZidTOCNewTemplate)
	DefaultHomeZidO    = MustParseO(api.ZidDefaultHome)
)

const maxZidO = 99999999999999

// ParseUintO interprets a string as a possible zettel identifier
// and returns its integer value.
func ParseUintO(s string) (uint64, error) {
	res, err := strconv.ParseUint(s, 10, 47)
	if err != nil {
		return 0, err
	}
	if res == 0 || res > maxZidO {
		return res, strconv.ErrRange
	}
	return res, nil
}

// ParseO interprets a string as a zettel identification and
// returns its value.
func ParseO(s string) (ZidO, error) {
	if len(s) != 14 {
		return InvalidO, strconv.ErrSyntax
	}
	res, err := ParseUintO(s)
	if err != nil {
		return InvalidO, err
	}
	return ZidO(res), nil
}

// MustParseO tries to interpret a string as a zettel identifier and returns
// its value or panics otherwise.
func MustParseO(s api.ZettelID) ZidO {
	zid, err := ParseO(string(s))
	if err == nil {
		return zid
	}
	panic(err)
}

// String converts the zettel identification to a string of 14 digits.
// Only defined for valid ids.
func (zid ZidO) String() string {
	var result [14]byte
	zid.toByteArray(&result)
	return string(result[:])
}

// ZettelID return the zettel identification as a api.ZettelID.
func (zid ZidO) ZettelID() api.ZettelID { return api.ZettelID(zid.String()) }

// Bytes converts the zettel identification to a byte slice of 14 digits.
// Only defined for valid ids.
func (zid ZidO) Bytes() []byte {
	var result [14]byte
	zid.toByteArray(&result)
	return result[:]
}

// toByteArray converts the Zid into a fixed byte array, usable for printing.
//
// Based on idea by Daniel Lemire: "Converting integers to fix-digit representations quickly"
// https://lemire.me/blog/2021/11/18/converting-integers-to-fix-digit-representations-quickly/
func (zid ZidO) toByteArray(result *[14]byte) {
	date := uint64(zid) / 1000000
	fullyear := date / 10000
	century, year := fullyear/100, fullyear%100
	monthday := date % 10000
	month, day := monthday/100, monthday%100
	time := uint64(zid) % 1000000
	hmtime, second := time/100, time%100
	hour, minute := hmtime/100, hmtime%100

	result[0] = byte(century/10) + '0'
	result[1] = byte(century%10) + '0'
	result[2] = byte(year/10) + '0'
	result[3] = byte(year%10) + '0'
	result[4] = byte(month/10) + '0'
	result[5] = byte(month%10) + '0'
	result[6] = byte(day/10) + '0'
	result[7] = byte(day%10) + '0'
	result[8] = byte(hour/10) + '0'
	result[9] = byte(hour%10) + '0'
	result[10] = byte(minute/10) + '0'
	result[11] = byte(minute%10) + '0'
	result[12] = byte(second/10) + '0'
	result[13] = byte(second%10) + '0'
}

// IsValid determines if zettel id is a valid one, e.g. consists of max. 14 digits.
func (zid ZidO) IsValid() bool { return 0 < zid && zid <= maxZidO }

// TimestampLayout to transform a date into a Zid and into other internal dates.
const TimestampLayout = "20060102150405"

// NewO returns a new zettel id based on the current time.
func NewO(withSeconds bool) ZidO {
	now := time.Now().Local()
	var s string
	if withSeconds {
		s = now.Format(TimestampLayout)
	} else {
		s = now.Format("20060102150400")
	}
	res, err := ParseO(s)
	if err != nil {
		panic(err)
	}
	return res
}
