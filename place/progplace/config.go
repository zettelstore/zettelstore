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
	"strings"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

func genConfigZettelM(zid id.Zid) *meta.Meta {
	if myPlace.startConfig == nil {
		return nil
	}
	m := meta.New(zid)
	m.Set(meta.KeyTitle, "Zettelstore Startup Configuration")
	m.Set(meta.KeyVisibility, meta.ValueVisibilityExpert)
	return m
}

func genConfigZettelC(m *meta.Meta) string {
	var sb strings.Builder
	for i, p := range myPlace.startConfig.Pairs(false) {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString("; ''")
		sb.WriteString(p.Key)
		sb.WriteString("''")
		if p.Value != "" {
			sb.WriteString("\n: ``")
			for _, r := range p.Value {
				if r == '`' {
					sb.WriteByte('\\')
				}
				sb.WriteRune(r)
			}
			sb.WriteString("``")
		}
	}
	return sb.String()
}
