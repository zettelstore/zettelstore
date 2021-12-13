//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package dirbox

import (
	"zettelstore.de/z/box/dirbox/notifydir"
	"zettelstore.de/z/box/dirbox/simpledir"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/logger"
)

func getDirSrvInfo(dirType string) (directoryServiceSpec, int, int) {
	for count := 0; count < 2; count++ {
		switch dirType {
		case kernel.BoxDirTypeNotify:
			return dirSrvNotify, 7, 1499
		case kernel.BoxDirTypeSimple:
			return dirSrvSimple, 1, 1
		default:
			dirType = kernel.Main.GetConfig(kernel.BoxService, kernel.BoxDefaultDirType).(string)
		}
	}
	panic("unable to set default dir box type: " + dirType)
}

func (dp *dirBox) setupDirService(log *logger.Logger) {
	switch dp.dirSrvSpec {
	case dirSrvSimple:
		dp.dirSrv = simpledir.NewService(log, dp.dir)
		dp.mustNotify = true
	default:
		dp.dirSrv = notifydir.NewService(log, dp.dir, dp.dirRescan, dp.cdata.Notify)
		dp.mustNotify = false
	}
}
