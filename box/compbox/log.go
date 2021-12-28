//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
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
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/kernel"
)

func genLogM(zid id.Zid) *meta.Meta {
	m := meta.New(zid)
	m.Set(api.KeyTitle, "Zettelstore Log")
	m.Set(api.KeyVisibility, api.ValueVisibilityExpert)
	return m
}
func genLogC(*meta.Meta) []byte {
	var buf bytes.Buffer
	buf.WriteString("|=No>|Timestamp|Level|Prefix|Message\n")
	for i, entry := range kernel.Main.RetrieveLogEntries() {
		fmt.Fprintf(&buf,
			"|%d|%v|%v|%v|%s\n",
			i+1, entry.TS.Format("2006-01-02 15:04:05.999999"),
			entry.Level, entry.Prefix, entry.Message)
	}
	return buf.Bytes()
}
