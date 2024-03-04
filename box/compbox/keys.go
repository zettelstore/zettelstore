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
	"bytes"
	"fmt"

	"zettelstore.de/client.fossil/api"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

func genKeysM(zid id.Zid) *meta.Meta {
	m := meta.New(zid)
	m.Set(api.KeyTitle, "Zettelstore Supported Metadata Keys")
	m.Set(api.KeyCreated, kernel.Main.GetConfig(kernel.CoreService, kernel.CoreVTime).(string))
	m.Set(api.KeyVisibility, api.ValueVisibilityLogin)
	return m
}

func genKeysC(*meta.Meta) []byte {
	keys := meta.GetSortedKeyDescriptions()
	var buf bytes.Buffer
	buf.WriteString("|=Name<|=Type<|=Computed?:|=Property?:\n")
	for _, kd := range keys {
		fmt.Fprintf(&buf,
			"|[[%v|query:%v?]]|%v|%v|%v\n", kd.Name, kd.Name, kd.Type.Name, kd.IsComputed(), kd.IsProperty())
	}
	return buf.Bytes()
}
