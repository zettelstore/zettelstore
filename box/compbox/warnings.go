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

	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

func genWarningsM(zid id.Zid) *meta.Meta {
	return getTitledMeta(zid, "Zettelstore Warnings")
}

func genWarningsC(ctx context.Context, cb *compBox) []byte {
	var buf bytes.Buffer
	buf.WriteString("* [[Zettel without stored creation date|query:created-missing:true]]\n")
	buf.WriteString("* [[Zettel with strange creation date|query:created<19700101000000]]\n")

	ws, err := cb.mapper.Warnings(ctx)
	if err != nil {
		buf.WriteString("**Error while fetching: ")
		buf.WriteString(err.Error())
		buf.WriteString("**\n")
		return buf.Bytes()
	}
	first := true
	ws.ForEach(func(zid id.Zid) {
		if first {
			first = false
			buf.WriteString("=== Mapper Warnings\n")
		}
		buf.WriteString("* [[")
		buf.WriteString(zid.String())
		buf.WriteString("]]\n")
	})

	return buf.Bytes()
}
