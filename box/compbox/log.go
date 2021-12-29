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

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/kernel"
)

func genLogM(zid id.Zid) *meta.Meta {
	m := meta.New(zid)
	m.Set(api.KeyTitle, "Zettelstore Log")
	m.Set(api.KeySyntax, api.ValueSyntaxText)
	return m
}

func genLogC(*meta.Meta) []byte {
	const tsFormat = "2006-01-02 15:04:05.999999"
	entries := kernel.Main.RetrieveLogEntries()
	var buf bytes.Buffer
	for _, entry := range entries {
		ts := entry.TS.Format(tsFormat)
		buf.WriteString(ts)
		for j := len(ts); j < len(tsFormat); j++ {
			buf.WriteByte('0')
		}
		buf.WriteByte(' ')
		buf.WriteString(entry.Level.Format())
		buf.WriteByte(' ')
		buf.WriteString(entry.Prefix)
		buf.WriteByte(' ')
		buf.WriteString(entry.Message)
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}
