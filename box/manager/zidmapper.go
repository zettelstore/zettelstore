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
	"maps"
	"sync"
	"time"

	"zettelstore.de/z/zettel/id"
)

// zidMapper transforms old-style zettel identifier (14 digits) into new one (4 alphanums).
//
// Since there are no new-style identifier defined, there is only support for old-style
// identifier by checking, whether they are suported as new-style or not.
//
// This will change in later versions.
type zidMapper struct {
	fetcher   zidfetcher
	defined   map[id.Zid]id.ZidN // predefined mapping, constant after creation
	mx        sync.RWMutex       // protect toNew ... nextZidN
	toNew     map[id.Zid]id.ZidN // working mapping old->new
	toOld     map[id.ZidN]id.Zid // working mapping new->old
	nextZidM  id.ZidN            // next zid for manual
	hadManual bool
	nextZidN  id.ZidN // next zid for normal zettel
}

type zidfetcher interface {
	fetchZids(context.Context) (*id.Set, error)
}

// NewZidMapper creates a new ZipMapper.
func NewZidMapper(fetcher zidfetcher) *zidMapper {
	defined := map[id.Zid]id.ZidN{
		id.Invalid: id.InvalidN,
		1:          id.MustParseN("0001"), // ZidVersion
		2:          id.MustParseN("0002"), // ZidHost
		3:          id.MustParseN("0003"), // ZidOperatingSystem
		4:          id.MustParseN("0004"), // ZidLicense
		5:          id.MustParseN("0005"), // ZidAuthors
		6:          id.MustParseN("0006"), // ZidDependencies
		7:          id.MustParseN("0007"), // ZidLog
		8:          id.MustParseN("0008"), // ZidMemory
		9:          id.MustParseN("0009"), // ZidSx
		10:         id.MustParseN("000a"), // ZidHTTP
		11:         id.MustParseN("000b"), // ZidAPI
		12:         id.MustParseN("000c"), // ZidWebUI
		13:         id.MustParseN("000d"), // ZidConsole
		20:         id.MustParseN("000e"), // ZidBoxManager
		21:         id.MustParseN("000f"), // ZidIndex
		22:         id.MustParseN("000g"), // ZidQuery
		90:         id.MustParseN("000h"), // ZidMetadataKey
		92:         id.MustParseN("000i"), // ZidParser
		96:         id.MustParseN("000j"), // ZidStartupConfiguration
		100:        id.MustParseN("000k"), // ZidRuntimeConfiguration
		101:        id.MustParseN("000l"), // ZidDirectory
		102:        id.MustParseN("000m"), // ZidWarnings
		10100:      id.MustParseN("000r"), // Base HTML Template
		10200:      id.MustParseN("000s"), // Login Form Template
		10300:      id.MustParseN("000t"), // List Zettel Template
		10401:      id.MustParseN("000u"), // Detail Template
		10402:      id.MustParseN("000v"), // Info Template
		10403:      id.MustParseN("000w"), // Form Template
		10404:      id.MustParseN("001z"), // Rename Form Template (will be removed in the future)
		10405:      id.MustParseN("000x"), // Delete Template
		10700:      id.MustParseN("000y"), // Error Template
		19000:      id.MustParseN("000p"), // Sxn Start Code
		19990:      id.MustParseN("000q"), // Sxn Base Code
		20001:      id.MustParseN("000z"), // Base CSS
		25001:      id.MustParseN("0010"), // User CSS
		40001:      id.MustParseN("000n"), // Generic Emoji
		59900:      id.MustParseN("000o"), // Sxn Prelude
		60010:      id.MustParseN("0011"), // zettel
		60020:      id.MustParseN("0012"), // confguration
		60030:      id.MustParseN("0013"), // role
		60040:      id.MustParseN("0014"), // tag
		90000:      id.MustParseN("0015"), // New Menu
		90001:      id.MustParseN("0016"), // New Zettel
		90002:      id.MustParseN("0017"), // New User
		90003:      id.MustParseN("0018"), // New Tag
		90004:      id.MustParseN("0019"), // New Role
		// 100000000,   // Manual               -> 0020-00yz
		9999999997:  id.MustParseN("00zx"), // ZidSession
		9999999998:  id.MustParseN("00zy"), // ZidAppDirectory
		9999999999:  id.MustParseN("00zz"), // ZidMapping
		10000000000: id.MustParseN("0100"), // ZidDefaultHome
	}
	toNew := maps.Clone(defined)
	toOld := make(map[id.ZidN]id.Zid, len(toNew))
	for o, n := range toNew {
		if _, found := toOld[n]; found {
			panic("duplicate predefined zid")
		}
		toOld[n] = o
	}

	return &zidMapper{
		fetcher:   fetcher,
		defined:   defined,
		toNew:     toNew,
		toOld:     toOld,
		nextZidM:  id.MustParseN("0020"),
		hadManual: false,
		nextZidN:  id.MustParseN("0101"),
	}
}

// isWellDefined returns true, if the given zettel identifier is predefined
// (as stated in the manual), or is part of the manual itself, or is greater than
// 19699999999999.
func (zm *zidMapper) isWellDefined(zid id.Zid) bool {
	if _, found := zm.defined[zid]; found || (1000000000 <= zid && zid <= 1099999999) {
		return true
	}
	if _, err := time.Parse("20060102150405", zid.String()); err != nil {
		return false
	}
	return 19700000000000 <= zid
}

// Warnings returns all zettel identifier with warnings.
func (zm *zidMapper) Warnings(ctx context.Context) (*id.Set, error) {
	allZids, err := zm.fetcher.fetchZids(ctx)
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

func (zm *zidMapper) GetZidN(zidO id.Zid) id.ZidN {
	zm.mx.RLock()
	if zidN, found := zm.toNew[zidO]; found {
		zm.mx.RUnlock()
		return zidN
	}
	zm.mx.RUnlock()

	zm.mx.Lock()
	defer zm.mx.Unlock()
	// Double check to avoid races
	if zidN, found := zm.toNew[zidO]; found {
		return zidN
	}

	if 1000000000 <= zidO && zidO <= 1099999999 {
		if zidO == 1000000000 {
			zm.hadManual = true
		}
		if zm.hadManual {
			zidN := zm.nextZidM
			zm.nextZidM++
			zm.toNew[zidO] = zidN
			zm.toOld[zidN] = zidO
			return zidN
		}
	}

	zidN := zm.nextZidN
	zm.nextZidN++
	zm.toNew[zidO] = zidN
	zm.toOld[zidN] = zidO
	return zidN
}

// OldToNewMapping returns the mapping of old format identifier to new format identifier.
func (zm *zidMapper) OldToNewMapping(ctx context.Context) (map[id.Zid]id.ZidN, error) {
	allZids, err := zm.fetcher.fetchZids(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[id.Zid]id.ZidN, allZids.Length())
	allZids.ForEach(func(zidO id.Zid) {
		zidN := zm.GetZidN(zidO)
		result[zidO] = zidN
	})
	return result, nil
}
