//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package compbox

import (
	"bytes"
	"fmt"

	"zettelstore.de/c/api"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

func genManagerM(zid id.Zid) *meta.Meta {
	m := meta.New(zid)
	m.Set(api.KeyTitle, "Zettelstore Box Manager")
	m.Set(api.KeyCreated, kernel.Main.GetConfig(kernel.CoreService, kernel.CoreStarted).(string))
	return m
}

func genManagerC(*meta.Meta) []byte {
	kvl := kernel.Main.GetServiceStatistics(kernel.BoxService)
	if len(kvl) == 0 {
		return nil
	}
	var buf bytes.Buffer
	buf.WriteString("|=Name|=Value>\n")
	for _, kv := range kvl {
		fmt.Fprintf(&buf, "| %v | %v\n", kv.Key, kv.Value)
	}
	return buf.Bytes()
}
