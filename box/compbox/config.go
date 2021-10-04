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
	"strings"

	"zettelstore.de/c/api"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
)

func genConfigZettelM(zid id.Zid) *meta.Meta {
	if myConfig == nil {
		return nil
	}
	m := meta.New(zid)
	m.Set(api.KeyTitle, "Zettelstore Startup Configuration")
	m.Set(api.KeyVisibility, api.ValueVisibilityExpert)
	return m
}

func genConfigZettelC(*meta.Meta) string {
	var sb strings.Builder
	for i, p := range myConfig.Pairs(false) {
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
