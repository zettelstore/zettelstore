//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package progplace provides zettel that inform the user about the internal Zettelstore state.
package progplace

import (
	"fmt"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/kernel"
)

func getVersionMeta(zid id.Zid, title string) *meta.Meta {
	m := meta.New(zid)
	m.Set(meta.KeyTitle, title)
	m.Set(meta.KeyVisibility, meta.ValueVisibilityExpert)
	return m
}

func genVersionBuildM(zid id.Zid) *meta.Meta {
	m := getVersionMeta(zid, "Zettelstore Version")
	m.Set(meta.KeyVisibility, meta.ValueVisibilityPublic)
	return m
}
func genVersionBuildC(*meta.Meta) string {
	return kernel.Main.GetConfig(kernel.CoreService, kernel.CoreVersion).(string)
}

func genVersionHostM(zid id.Zid) *meta.Meta {
	return getVersionMeta(zid, "Zettelstore Host")
}
func genVersionHostC(*meta.Meta) string {
	return kernel.Main.GetConfig(kernel.CoreService, kernel.CoreHostname).(string)
}

func genVersionOSM(zid id.Zid) *meta.Meta {
	return getVersionMeta(zid, "Zettelstore Operating System")
}
func genVersionOSC(*meta.Meta) string {
	return fmt.Sprintf(
		"%v/%v",
		kernel.Main.GetConfig(kernel.CoreService, kernel.CoreGoOS).(string),
		kernel.Main.GetConfig(kernel.CoreService, kernel.CoreGoArch).(string),
	)
}
