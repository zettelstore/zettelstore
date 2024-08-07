//-----------------------------------------------------------------------------
// Copyright (c) 2024-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2024-present Detlef Stern
//-----------------------------------------------------------------------------

package compbox

import (
	"bytes"
	"context"

	"t73f.de/r/zsc/api"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

// Zettelstore Identifier Mapping.
//
// In the first stage of migration process, it is a computed zettel showing a
// hypothetical mapping. In later stages, it will be stored as a normal zettel
// that is updated when a new zettel is created or an old zettel is deleted.

func genMappingM(zid id.Zid) *meta.Meta {
	m := getTitledMeta(zid, "Zettelstore Identifier Mapping View (TEMP for v0.19-dev)")
	m.Set(api.KeySyntax, meta.SyntaxText)
	return m
}

func genMappingC(ctx context.Context, cb *compBox) []byte {
	var buf bytes.Buffer
	toNew, err := cb.mapper.OldToNewMapping(ctx)
	if err != nil {
		buf.WriteString("**Error while fetching: ")
		buf.WriteString(err.Error())
		buf.WriteString("**\n")
		return buf.Bytes()
	}
	oldZids := id.NewSetCap(len(toNew))
	for zidO := range toNew {
		oldZids.Add(zidO)
	}
	oldZids.ForEach(func(zidO id.Zid) {
		buf.WriteString(zidO.String())
		buf.WriteByte(' ')
		buf.WriteString(toNew[zidO].String())
		buf.WriteByte('\n')
	})
	return buf.Bytes()
}
