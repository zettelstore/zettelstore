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
	"zettelstore.de/z/place/dirplace/notifydir"
	"zettelstore.de/z/place/dirplace/simpledir"
	"zettelstore.de/z/service"
)

func getDirSrvInfo(dirType string) (directoryServiceSpec, int, int) {
	for count := 0; count < 2; count++ {
		switch dirType {
		case service.PlaceDirTypeNotify:
			return dirSrvNotify, 7, 1499
		case service.PlaceDirTypeSimple:
			return dirSrvSimple, 1, 1
		default:
			dirType = service.Main.GetConfig(service.SubPlace, service.PlaceDefaultDirType).(string)
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
