//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
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
	"bytes"
	"fmt"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

func genKeysM(zid id.Zid) *meta.Meta {
	m := meta.New(zid)
	m.Set(api.KeyTitle, "Zettelstore Supported Metadata Keys")
	m.Set(api.KeyVisibility, api.ValueVisibilityLogin)
	return m
}

func genKeysC(*meta.Meta) []byte {
	keys := meta.GetSortedKeyDescriptions()
	var buf bytes.Buffer
	buf.WriteString("|=Name<|=Type<|=Computed?:|=Property?:\n")
	for _, kd := range keys {
		fmt.Fprintf(&buf,
			"|%v|%v|%v|%v\n", kd.Name, kd.Type.Name, kd.IsComputed(), kd.IsProperty())
	}
	return buf.Bytes()
}
