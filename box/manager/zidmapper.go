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

package manager

import (
	"context"

	"zettelstore.de/z/zettel/id"
)

// zidMapper transforms old-style zettel identifier (14 digits) into new one (4 alphanums).
//
// Since there are no new-style identifier defined, there is only support for old-style
// identifier by checking, whether they are suported as new-style or not.
//
// This will change in later versions.
type zidMapper struct {
	fetcher zidfetcher
	defined *id.Set // supported old-style identifier
}

type zidfetcher interface {
	FetchZids(context.Context) (*id.Set, error)
}

// NewZidMapper creates a new ZipMapper.
func NewZidMapper(fetcher zidfetcher) *zidMapper {
	defined := id.NewSet(
		id.Invalid,
		1,     // ZidVersion           -> 0001
		2,     // ZidHost              -> 0002
		3,     // ZidOperatingSystem   -> 0003
		4,     // ZidLicense           -> 0004
		5,     // ZidAuthors           -> 0005
		6,     // ZidDependencies      -> 0006
		7,     // ZidLog               -> 0007
		8,     // ZidMemory            -> 0008
		9,     // ZidSx                -> 0009
		10,    // ZidHTTP              -> 000a
		11,    // ZidAPI               -> 000b
		12,    // ZidWebUI             -> 000c
		13,    // ZidConsole           -> 000d
		20,    // ZidBoxManager        -> 000e
		21,    // ZidIndex             -> 000f
		22,    // ZidQuery             -> 000g
		90,    // ZidMetadataKey       -> 000h
		92,    // ZidParser            -> 000i
		96,    // ZidStartupConfig     -> 000j
		100,   // ZidRuntimeConfig     -> 000k
		101,   // ZidDirectory         -> 000k
		102,   // ZidWarnings          -> 000m
		10100, // Base HTML Template   -> 000r
		10200, // Login Form Template  -> 000s
		10300, // List Zettel Template -> 000t
		10401, // Detail Template      -> 000u
		10402, // Info Template        -> 000v
		10403, // Form Template        -> 000w
		10404, // Rename Form Template (will be removed in the future)
		10405, // Delete Template      -> 000x
		10700, // Error Template       -> 000y
		19000, // Sxn Start Code       -> 000p
		19990, // Sxn Base Code        -> 000q
		20001, // Base CSS             -> 000z
		25001, // User CSS             -> 0010
		40001, // Generic Emoji        -> 000n
		59900, // Sxn Prelude          -> 000o
		60010, // zettel               -> 0011
		60020, // confguration         -> 0012
		60030, // role                 -> 0013
		60040, // tag                  -> 0014
		90000, // New Menu             -> 0015
		90001, // New Zettel           -> 0016
		90002, // New User             -> 0017
		90003, // New Tag              -> 0018
		90004, // New Role             -> 0019
		// 100000000,   // Manual               -> 0020-00yz
		9999999998,  // ZidAppDirectory      -> 00zy
		9999999999,  // ZidMapping           -> 00zz
		10000000000, // ZidDefaultHome       -> 0100
	)
	return &zidMapper{
		fetcher: fetcher,
		defined: defined,
	}
}

// isWellDefined returns true, if the given zettel identifier is predefined
// (as stated in the manual), or is part of the manual itself, or is greater than
// 19699999999999.
func (zm *zidMapper) isWellDefined(zid id.Zid) bool {
	if 19700000000000 <= zid {
		return true
	}
	if 1000000000 <= zid && zid <= 1999999999 {
		return true
	}
	return zm.defined.Contains(zid)
}

// Warnings returns all zettel identifier with warnings.
func (zm *zidMapper) Warnings(ctx context.Context) (*id.Set, error) {
	allZids, err := zm.fetcher.FetchZids(ctx)
	if err != nil {
		return nil, err
	}
	warnings := id.NewSet()
	allZids.ForEach(func(zid id.Zid) {
		if !zm.isWellDefined(zid) {
			warnings = warnings.Add(zid)
		}
	})
	return warnings, nil
}
