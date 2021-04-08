//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// +build darwin

// Package dirplace provides a directory-based zettel place.
package dirplace

func (dp *dirPlace) setupDirService() {
	dp.dirSrv = plaindir.NewService(dp.dir, dp.dirRescan, dp.cdata.Notify)
	dp.mustNotify = true
}
