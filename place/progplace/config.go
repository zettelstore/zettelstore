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
	m.Set(meta.KeyTitle, "Zettelstore Startup Values")
	m.Set(meta.KeyRole, meta.ValueRoleConfiguration)
	m.Set(meta.KeySyntax, meta.ValueSyntaxZmk)
	m.Set(meta.KeyVisibility, meta.ValueVisibilitySimple)
	m.Set(meta.KeyReadOnly, meta.ValueTrue)
	return m
}

func genConfigC(m *meta.Meta) string {
	var sb strings.Builder
	sb.WriteString("|=Name|=Value>\n")
	fmt.Fprintf(&sb, "|Simple|%v\n", startup.IsSimple())
	fmt.Fprintf(&sb, "|Verbose|%v\n", startup.IsVerbose())
	fmt.Fprintf(&sb, "|Read-only|%v\n", startup.IsReadOnlyMode())
	fmt.Fprintf(&sb, "|URL prefix|%v\n", startup.URLPrefix())
	// There must be a space before the next "%v". Listen address may start with a ":"
	fmt.Fprintf(&sb, "|Listen address| %v\n", service.Main.GetConfig(service.SubWeb, service.WebListenAddress))
	fmt.Fprintf(&sb, "|Authentication enabled|%v\n", startup.WithAuth())
	fmt.Fprintf(&sb, "|Secure cookie|%v\n", startup.SecureCookie())
	fmt.Fprintf(&sb, "|Persistent Cookie|%v\n", startup.PersistentCookie())
	html, api := startup.TokenLifetime()
	fmt.Fprintf(&sb, "|API Token lifetime|%v\n", api)
	fmt.Fprintf(&sb, "|HTML Token lifetime|%v\n", html)
	fmt.Fprintf(&sb, "|Default directory place type|%v", startup.DefaultDirPlaceType())
	return sb.String()
}
