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
	"zettelstore.de/z/place"
)

func genManagerM(zid id.Zid) *meta.Meta {
	if myPlace.manager == nil {
		return nil
	}
	m := meta.New(zid)
	m.Set(meta.KeyTitle, "Zettelstore Place Manager")
	return m
}

func genManagerC(*meta.Meta) string {
	mgr := myPlace.manager

	var stats place.Stats
	mgr.ReadStats(&stats)

	var sb strings.Builder
	sb.WriteString("|=Name|=Value>\n")
	fmt.Fprintf(&sb, "|Read-only| %v\n", stats.ReadOnly)
	fmt.Fprintf(&sb, "|Zettel| %v\n", stats.Zettel)
	fmt.Fprintf(&sb, "|Sub-places| %v\n", mgr.NumPlaces())
	return sb.String()
}
