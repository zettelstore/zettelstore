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

	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/service"
)

func genConfigZettelM(zid id.Zid) *meta.Meta {
	if myPlace.startConfig == nil {
		return nil
	}
	m := meta.New(zid)
	m.Set(meta.KeyTitle, "Zettelstore Startup Configuration")
	m.Set(meta.KeyVisibility, meta.ValueVisibilitySimple)
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

func genConfigM(zid id.Zid) *meta.Meta {
	if myPlace.startConfig == nil {
		return nil
	}
	m := meta.New(zid)
	m.Set(meta.KeyTitle, "Zettelstore Service Configuration")
	m.Set(meta.KeyRole, meta.ValueRoleConfiguration)
	m.Set(meta.KeySyntax, meta.ValueSyntaxZmk)
	m.Set(meta.KeyVisibility, meta.ValueVisibilitySimple)
	m.Set(meta.KeyReadOnly, meta.ValueTrue)
	return m
}

func genConfigC(m *meta.Meta) string {
	var sb strings.Builder
	sb.WriteString("|=Name|=Value>\n")
	fmt.Fprintf(&sb, "|Authentication enabled|%v\n", startup.WithAuth())
	fmt.Fprintf(&sb, "|Secure cookie|%v\n", startup.SecureCookie())
	fmt.Fprintf(&sb, "|Persistent Cookie|%v\n", startup.PersistentCookie())
	html, api := startup.TokenLifetime()
	fmt.Fprintf(&sb, "|API Token lifetime|%v\n", api)
	fmt.Fprintf(&sb, "|HTML Token lifetime|%v\n", html)
	writeSubsrvConfig(&sb, service.SubMain, "Main")
	writeSubsrvConfig(&sb, service.SubAuth, "Authentication")
	writeSubsrvConfig(&sb, service.SubPlace, "Zettel places")
	writeSubsrvConfig(&sb, service.SubWeb, "Web")
	return sb.String()
}

func writeSubsrvConfig(sbp *strings.Builder, subsrv service.Subservice, name string) {
	configList := service.Main.GetConfigList(subsrv)
	if len(configList) == 0 {
		return
	}
	fmt.Fprintln(sbp, "===", name)
	sbp.WriteString("|=Key|=Description|=Value>\n")
	for _, config := range configList {
		fmt.Fprintf(sbp, "|%v| %v| %v\n", config.Key, config.Descr, config.Value)
	}
}
