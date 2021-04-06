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

import (
	"time"

	"zettelstore.de/z/place/change"
	"zettelstore.de/z/place/dirplace/directory"
	"zettelstore.de/z/place/dirplace/notifydir"
)

func makeDirService(dir string, dirRescan time.Duration, notify chan<- change.Info) directory.Service {
	return notifydir.NewService(dir, dirRescan, notify)
}
