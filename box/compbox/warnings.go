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

package compbox

import (
	"bytes"

	"t73f.de/r/zsc/api"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

func genWarningsM(zid id.Zid) *meta.Meta {
	m := getVersionMeta(zid, "Zettelstore Warnings")
	m.Set(api.KeyCreated, kernel.Main.GetConfig(kernel.CoreService, kernel.CoreStarted).(string))
	m.Set(api.KeyVisibility, api.ValueVisibilityExpert)
	return m
}
func genWarningsC(*meta.Meta) []byte {
	var buf bytes.Buffer
	buf.WriteString("* [[Zettel without stored creation date|query:created-missing:true]]\n")
	buf.WriteString("* [[Zettel with strange creation date|query:created-missing:true]]\n")
	return buf.Bytes()
}
