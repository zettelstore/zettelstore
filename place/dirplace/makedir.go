//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package dirplace provides a directory-based zettel place.
package dirplace

import (
	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/place/dirplace/notifydir"
	"zettelstore.de/z/place/dirplace/simpledir"
)

func getDirSrvInfo(dirType string) (directoryServiceSpec, int, int) {
	for count := 0; count < 2; count++ {
		switch dirType {
		case startup.ValueDirPlaceTypeNotify:
			return dirSrvNotify, 7, 1499
		case startup.ValueDirPlaceTypeSimple:
			return dirSrvSimple, 1, 1
		default:
			dirType = startup.DefaultDirPlaceType()
		}
	}
	panic("unable to set default dir place type: " + dirType)
}

func (dp *dirPlace) setupDirService() {
	switch dp.dirSrvSpec {
	case dirSrvSimple:
		dp.dirSrv = simpledir.NewService(dp.dir)
		dp.mustNotify = true
	default:
		dp.dirSrv = notifydir.NewService(dp.dir, dp.dirRescan, dp.cdata.Notify)
		dp.mustNotify = false
	}
}
