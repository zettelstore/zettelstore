//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package progplace provides zettel that inform the user about the internal
// Zettelstore state.
package progplace

import (
	"fmt"
	"strings"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/index"
)

func genIndexerM(zid id.Zid) *meta.Meta {
	if myPlace.indexer == nil {
		return nil
	}
	m := meta.New(zid)
	m.Set(meta.KeyTitle, "Zettelstore Indexer")
	return m
}

func genIndexerC(*meta.Meta) string {
	ixer := myPlace.indexer

	var stats index.IndexerStats
	ixer.ReadStats(&stats)

	var sb strings.Builder
	sb.WriteString("|=Name|=Value>\n")
	fmt.Fprintf(&sb, "|Zettel| %v\n", stats.Store.Zettel)
	fmt.Fprintf(&sb, "|Last re-index| %v\n", stats.LastReload.Format("2006-01-02 15:04:05 -0700 MST"))
	fmt.Fprintf(&sb, "|Indexes since last re-index| %v\n", stats.IndexesSinceReload)
	fmt.Fprintf(&sb, "|Duration last index| %vms\n", stats.DurLastIndex.Milliseconds())
	fmt.Fprintf(&sb, "|Zettel enrichments| %v\n", stats.Store.Updates)
	fmt.Fprintf(&sb, "|Indexed words| %v\n", stats.Store.Words)
	return sb.String()
}
