//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// +build linux windows

// Package dirplace provides a directory-based zettel place.
package dirplace

import "zettelstore.de/z/place/dirplace/notifydir"

func (dp *dirPlace) setupDirService() {
	switch dp.dirSrvType {
	default:
		dp.dirSrv = notifydir.NewService(dp.dir, dp.dirRescan, dp.cdata.Notify)
		dp.mustNotify = false
	}
}
