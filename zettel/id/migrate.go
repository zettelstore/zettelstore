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
	defined, workset map[ZidO]Zid
	lastZidO         ZidO
	nextZid          Zid
	ranges           []zidRange
	usedZids         map[Zid]struct{}
}

type zidRange struct {
	lowO, highO ZidO
	base        Zid
}

// NewZidMigrator creates a new zid migrator.
func NewZidMigrator() *ZidMigrator {
	defined := map[ZidO]Zid{
		0:               0,                 // Invalid
		1:               MustParse("0001"), // Zettelstore Version
		2:               MustParse("0002"), // Zettelstore Host
		3:               MustParse("0003"), // Zettelstore Operating System
		4:               MustParse("0004"), // Zettelstore License
		5:               MustParse("0005"), // Zettelstore Contributors
		6:               MustParse("0006"), // Zettelstore Dependencies
		7:               MustParse("0007"), // Zettelstore Log
		8:               MustParse("0008"), // Zettelstore Memory
		20:              MustParse("000g"), // Zettelstore Box Manager
		90:              MustParse("000t"), // Zettelstore Supported Metadata Keys
		92:              MustParse("000v"), // Zettelstore Supported Parser
		96:              MustParse("000x"), // Zettelstore Startup Configuration
		100:             MustParse("000z"), // Zettelstore Runtime Configuration
		10100:           MustParse("0010"), // Zettelstore Base HTML Template
		10200:           MustParse("0011"), // Zettelstore Login Form HTML Template
		10300:           MustParse("0012"), // Zettelstore List Zettel HTML Template
		10401:           MustParse("0013"), // Zettelstore Detail HTML Template
		10402:           MustParse("0014"), // Zettelstore Info HTML Template
		10403:           MustParse("0015"), // Zettelstore Form HTML Template
		10404:           MustParse("0016"), // Zettelstore Rename Form HTML Template
		10405:           MustParse("0017"), // Zettelstore Delete HTML Template
		10700:           MustParse("0018"), // Zettelstore Error HTML Template
		19000:           MustParse("0021"), // Zettelstore Sxn Start Code
		19990:           MustParse("0022"), // Zettelstore Sxn Base Code
		20001:           MustParse("0030"), // Zettelstore Base CSS
		25001:           MustParse("0031"), // Zettelstore User CSS
		40001:           MustParse("0032"), // Generic Emoji
		59900:           MustParse("0020"), // Zettelstore Sxn Prelude
		60010:           MustParse("0041"), // zettel
		60020:           MustParse("0042"), // confguration
		60030:           MustParse("0043"), // role
		60040:           MustParse("0044"), // tag
		90000:           MustParse("0050"), // New Menu
		90001:           MustParse("0051"), // New Zettel
		90002:           MustParse("0052"), // New User
		90003:           MustParse("0053"), // New Tag
		90004:           MustParse("0054"), // New Role
		100000000:       MustParse("0100"), // Zettelstore Manual (bis 02zz)
		200000000:       MustParse("0300"), // Reserviert (bis 0tzz)
		9000000000:      MustParse("0u00"), // Externe Anwendungen (bis 0zzz)
		DefaultHomeZidO: MustParse("1000"), // Default home zettel
	}
	usedZids := make(map[Zid]struct{}, len(defined))
	for _, zid := range defined {
		if _, found := usedZids[zid]; found {
			panic("duplicate predefined zid")
		}
		usedZids[zid] = struct{}{}
	}
	return &ZidMigrator{
		defined:  defined,
		workset:  maps.Clone(defined),
		lastZidO: InvalidO,
		nextZid:  MustParse("1001"),
		ranges: []zidRange{
			{10000, 19999, MustParse("0010")},
			{20000, 29999, MustParse("0030")},
			{40000, 49999, MustParse("0032")},
			{50000, 59999, MustParse("0020")},
			{60000, 69999, MustParse("0040")},
			{90000, 99999, MustParse("0050")},
		},
		usedZids: usedZids,
	}
}

// Migrate an old Zid to a new one.
//
// Old zids must increase.
func (zm *ZidMigrator) Migrate(zidO ZidO) (Zid, error) {
	if zid, found := zm.workset[zidO]; found {
		return zid, nil
	}
	if zidO <= zm.lastZidO {
		return Invalid, fmt.Errorf("out of sequence: %v", zidO)
	}
	zm.lastZidO = zidO
	if (zidO < 10000) ||
		(30000 <= zidO && zidO < 40000) ||
		(70000 <= zidO && zidO < 90000) ||
		(100000 <= zidO && zidO < 100000000) ||
		(200000000 <= zidO && zidO < 9000001000) ||
		(9000002000 <= zidO && zidO < DefaultHomeZidO) {
		return 0, fmt.Errorf("old Zid out of supported range: %v", zidO)
	}
	if DefaultHomeZidO < zidO {
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
	return Invalid, nil
}

func (zm *ZidMigrator) retrieveNextInRange(lowO, highO ZidO) Zid {
	var currentMax Zid
	for zidO, zid := range zm.workset {
		if lowO <= zidO && zidO <= highO && currentMax < zid {
			currentMax = zid
		}
	}
	return currentMax + 1
}

func (zm *ZidMigrator) checkZid(zid Zid) (Zid, error) {
	if _, found := zm.usedZids[zid]; found {
		return Invalid, fmt.Errorf("zid %v alredy used", zid)
	}
	zm.usedZids[zid] = struct{}{}
	return zid, nil
}
