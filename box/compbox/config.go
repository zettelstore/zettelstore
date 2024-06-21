//-----------------------------------------------------------------------------
// Copyright (c) 2020-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2020-present Detlef Stern
//-----------------------------------------------------------------------------

package compbox

import (
	"bytes"

	"t73f.de/r/zsc/api"
	"zettelstore.de/z/kernel"
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

func genConfigZettelM(zid id.ZidO) *meta.Meta {
	if myConfig == nil {
		return nil
	}
	m := meta.New(zid)
	m.Set(api.KeyTitle, "Zettelstore Startup Configuration")
	m.Set(api.KeyCreated, kernel.Main.GetConfig(kernel.CoreService, kernel.CoreStarted).(string))
	m.Set(api.KeyVisibility, api.ValueVisibilityExpert)
	return m
}

func genConfigZettelC(*meta.Meta) []byte {
	var buf bytes.Buffer
	for i, p := range myConfig.Pairs() {
		if i > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString("; ''")
		buf.WriteString(p.Key)
		buf.WriteString("''")
		if p.Value != "" {
			buf.WriteString("\n: ``")
			for _, r := range p.Value {
				if r == '`' {
					buf.WriteByte('\\')
				}
				buf.WriteRune(r)
			}
			buf.WriteString("``")
		}
	}
	return buf.Bytes()
}
