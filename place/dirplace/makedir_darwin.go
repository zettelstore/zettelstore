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

import "zettelstore.de/z/place/dirplace/simpledir"

func (dp *dirPlace) setupDirService() {
	dp.dirSrv = simpledir.NewService(dp.dir)
	dp.mustNotify = true
}
