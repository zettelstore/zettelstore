//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package compbox provides zettel that have computed content.
package compbox

import (
	"fmt"
	"strings"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/kernel"
)

func genManagerM(zid id.Zid) *meta.Meta {
	m := meta.New(zid)
	m.Set(meta.KeyTitle, "Zettelstore Box Manager")
	return m
}

func genManagerC(*meta.Meta) string {
	kvl := kernel.Main.GetServiceStatistics(kernel.BoxService)
	if len(kvl) == 0 {
		return "No statistics available"
	}
	var sb strings.Builder
	sb.WriteString("|=Name|=Value>\n")
	for _, kv := range kvl {
		fmt.Fprintf(&sb, "| %v | %v\n", kv.Key, kv.Value)
	}
	return sb.String()
}
