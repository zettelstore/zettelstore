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

package id

import (
	"fmt"
	"maps"
)

// This is for migration of Zid0 to Zid.

// ZidMigrator does the actual migration.
type ZidMigrator struct {
	defined, workset map[Zid]ZidN
	lastZid          Zid
	nextZid          ZidN
	ranges           []zidRange
	usedZids         map[ZidN]struct{}
}

type zidRange struct {
	lowO, highO Zid
	base        ZidN
}

// NewZidMigrator creates a new zid migrator.
func NewZidMigrator() *ZidMigrator {
	defined := map[Zid]ZidN{
		0:              0,                  // Invalid
		1:              MustParseN("0001"), // Zettelstore Version
		2:              MustParseN("0002"), // Zettelstore Host
		3:              MustParseN("0003"), // Zettelstore Operating System
		4:              MustParseN("0004"), // Zettelstore License
		5:              MustParseN("0005"), // Zettelstore Contributors
		6:              MustParseN("0006"), // Zettelstore Dependencies
		7:              MustParseN("0007"), // Zettelstore Log
		8:              MustParseN("0008"), // Zettelstore Memory
		20:             MustParseN("000g"), // Zettelstore Box Manager
		90:             MustParseN("000t"), // Zettelstore Supported Metadata Keys
		92:             MustParseN("000v"), // Zettelstore Supported Parser
		96:             MustParseN("000x"), // Zettelstore Startup Configuration
		100:            MustParseN("000z"), // Zettelstore Runtime Configuration
		10100:          MustParseN("0010"), // Zettelstore Base HTML Template
		10200:          MustParseN("0011"), // Zettelstore Login Form HTML Template
		10300:          MustParseN("0012"), // Zettelstore List Zettel HTML Template
		10401:          MustParseN("0013"), // Zettelstore Detail HTML Template
		10402:          MustParseN("0014"), // Zettelstore Info HTML Template
		10403:          MustParseN("0015"), // Zettelstore Form HTML Template
		10404:          MustParseN("0016"), // Zettelstore Rename Form HTML Template
		10405:          MustParseN("0017"), // Zettelstore Delete HTML Template
		10700:          MustParseN("0018"), // Zettelstore Error HTML Template
		19000:          MustParseN("0021"), // Zettelstore Sxn Start Code
		19990:          MustParseN("0022"), // Zettelstore Sxn Base Code
		20001:          MustParseN("0030"), // Zettelstore Base CSS
		25001:          MustParseN("0031"), // Zettelstore User CSS
		40001:          MustParseN("0032"), // Generic Emoji
		59900:          MustParseN("0020"), // Zettelstore Sxn Prelude
		60010:          MustParseN("0041"), // zettel
		60020:          MustParseN("0042"), // confguration
		60030:          MustParseN("0043"), // role
		60040:          MustParseN("0044"), // tag
		90000:          MustParseN("0050"), // New Menu
		90001:          MustParseN("0051"), // New Zettel
		90002:          MustParseN("0052"), // New User
		90003:          MustParseN("0053"), // New Tag
		90004:          MustParseN("0054"), // New Role
		100000000:      MustParseN("0100"), // Zettelstore Manual (bis 02zz)
		200000000:      MustParseN("0300"), // Reserviert (bis 0tzz)
		9000000000:     MustParseN("0u00"), // Externe Anwendungen (bis 0zzz)
		DefaultHomeZid: MustParseN("1000"), // Default home zettel
	}
	usedZids := make(map[ZidN]struct{}, len(defined))
	for _, zid := range defined {
		if _, found := usedZids[zid]; found {
			panic("duplicate predefined zid")
		}
		usedZids[zid] = struct{}{}
	}
	return &ZidMigrator{
		defined: defined,
		workset: maps.Clone(defined),
		lastZid: Invalid,
		nextZid: MustParseN("1001"),
		ranges: []zidRange{
			{10000, 19999, MustParseN("0010")},
			{20000, 29999, MustParseN("0030")},
			{40000, 49999, MustParseN("0032")},
			{50000, 59999, MustParseN("0020")},
			{60000, 69999, MustParseN("0040")},
			{90000, 99999, MustParseN("0050")},
		},
		usedZids: usedZids,
	}
}

// Migrate an old Zid to a new one.
//
// Old zids must increase.
func (zm *ZidMigrator) Migrate(zidO Zid) (ZidN, error) {
	if zid, found := zm.workset[zidO]; found {
		return zid, nil
	}
	if zidO <= zm.lastZid {
		return InvalidN, fmt.Errorf("out of sequence: %v", zidO)
	}
	zm.lastZid = zidO
	if (zidO < 10000) ||
		(30000 <= zidO && zidO < 40000) ||
		(70000 <= zidO && zidO < 90000) ||
		(100000 <= zidO && zidO < 100000000) ||
		(200000000 <= zidO && zidO < 9000001000) ||
		(9000002000 <= zidO && zidO < DefaultHomeZid) {
		return 0, fmt.Errorf("old Zid out of supported range: %v", zidO)
	}
	if DefaultHomeZid < zidO {
		zid := zm.nextZid
		zm.nextZid++
		zm.workset[zidO] = zid
		return zm.checkZid(zid)
	}
	for _, zr := range zm.ranges {
		if zidO < zr.lowO || zr.highO < zidO {
			continue
		}
		zid := zm.retrieveNextInRange(zr.lowO, zr.highO)
		zm.workset[zidO] = zid
		return zm.checkZid(zid)
	}
	return InvalidN, nil
}

func (zm *ZidMigrator) retrieveNextInRange(lowO, highO Zid) ZidN {
	var currentMax ZidN
	for zidO, zid := range zm.workset {
		if lowO <= zidO && zidO <= highO && currentMax < zid {
			currentMax = zid
		}
	}
	return currentMax + 1
}

func (zm *ZidMigrator) checkZid(zid ZidN) (ZidN, error) {
	if _, found := zm.usedZids[zid]; found {
		return InvalidN, fmt.Errorf("zid %v alredy used", zid)
	}
	zm.usedZids[zid] = struct{}{}
	return zid, nil
}
