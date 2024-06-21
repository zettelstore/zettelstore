//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2020-present Detlef Stern
//-----------------------------------------------------------------------------

package compbox

import (
	"t73f.de/r/zsc/api"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

func getVersionMeta(zid id.ZidO, title string) *meta.Meta {
	m := meta.New(zid)
	m.Set(api.KeyTitle, title)
	m.Set(api.KeyVisibility, api.ValueVisibilityExpert)
	return m
}

func genVersionBuildM(zid id.ZidO) *meta.Meta {
	m := getVersionMeta(zid, "Zettelstore Version")
	m.Set(api.KeyCreated, kernel.Main.GetConfig(kernel.CoreService, kernel.CoreVTime).(string))
	m.Set(api.KeyVisibility, api.ValueVisibilityLogin)
	return m
}
func genVersionBuildC(*meta.Meta) []byte {
	return []byte(kernel.Main.GetConfig(kernel.CoreService, kernel.CoreVersion).(string))
}

func genVersionHostM(zid id.ZidO) *meta.Meta {
	m := getVersionMeta(zid, "Zettelstore Host")
	m.Set(api.KeyCreated, kernel.Main.GetConfig(kernel.CoreService, kernel.CoreStarted).(string))
	return m
}
func genVersionHostC(*meta.Meta) []byte {
	return []byte(kernel.Main.GetConfig(kernel.CoreService, kernel.CoreHostname).(string))
}

func genVersionOSM(zid id.ZidO) *meta.Meta {
	m := getVersionMeta(zid, "Zettelstore Operating System")
	m.Set(api.KeyCreated, kernel.Main.GetConfig(kernel.CoreService, kernel.CoreStarted).(string))
	return m
}
func genVersionOSC(*meta.Meta) []byte {
	goOS := kernel.Main.GetConfig(kernel.CoreService, kernel.CoreGoOS).(string)
	goArch := kernel.Main.GetConfig(kernel.CoreService, kernel.CoreGoArch).(string)
	result := make([]byte, 0, len(goOS)+len(goArch)+1)
	result = append(result, goOS...)
	result = append(result, '/')
	return append(result, goArch...)
}
