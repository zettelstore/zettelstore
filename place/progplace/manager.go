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

	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/place"
)

func genManagerM(zid id.Zid) *meta.Meta {
	m := meta.New(zid)
	m.Set(meta.KeyTitle, "Zettelstore Place Manager")
	return m
}

func genManagerC(*meta.Meta) string {
	mgr := startup.PlaceManager()

	var mStats place.Stats
	mgr.ReadStats(&mStats)

	var sb strings.Builder
	sb.WriteString("|=Name|=Value>\n")
	fmt.Fprintf(&sb, "|Read-only| %v\n", mStats.ReadOnly)
	fmt.Fprintf(&sb, "|Sub-places| %v\n", mgr.NumPlaces())
	fmt.Fprintf(&sb, "|Zettel (total)| %v\n", mStats.ZettelTotal)
	fmt.Fprintf(&sb, "|Zettel (indexable)| %v\n", mStats.ZettelIndexed)
	fmt.Fprintf(&sb, "|Last re-index| %v\n", mStats.LastReload.Format("2006-01-02 15:04:05 -0700 MST"))
	fmt.Fprintf(&sb, "|Indexes since last re-index| %v\n", mStats.IndexesSinceReload)
	fmt.Fprintf(&sb, "|Duration last index| %vms\n", mStats.DurLastIndex.Milliseconds())
	fmt.Fprintf(&sb, "|Indexed words| %v\n", mStats.IndexedWords)
	fmt.Fprintf(&sb, "|Indexed URLs| %v\n", mStats.IndexedUrls)
	fmt.Fprintf(&sb, "|Zettel enrichments| %v\n", mStats.IndexUpdates)
	return sb.String()
}
