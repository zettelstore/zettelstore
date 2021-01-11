//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package progplace provides zettel that inform the user about the internal Zettelstore state.
package progplace

import (
	"fmt"
	"strings"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

func genKeysM(zid id.Zid) *meta.Meta {
	m := meta.New(zid)
	m.Set(meta.KeyTitle, "Zettelstore Supported Metadata Keys")
	m.Set(meta.KeyVisibility, meta.ValueVisibilityLogin)
	return m
}

func genKeysC(*meta.Meta) string {
	keys := meta.GetSortedKeyDescriptions()
	var sb strings.Builder
	sb.WriteString("|=Name<|=Type<|=Computed?:|=Property?:\n")
	for _, kd := range keys {
		fmt.Fprintf(&sb,
			"|%v|%v|%v|%v\n", kd.Name, kd.Type.Name, kd.IsComputed(), kd.IsProperty())
	}
	return sb.String()
}
