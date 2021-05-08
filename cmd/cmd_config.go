//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package cmd

import (
	"flag"
	"fmt"

	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/domain/meta"
	"zettelstore.de/z/service"
)

// ---------- Subcommand: config ---------------------------------------------

func cmdConfig(*flag.FlagSet, *meta.Meta) (int, error) {
	srvm := service.Main
	fmt.Println("Web")
	fmt.Printf("  Listen address    = %q\n", srvm.GetConfig(service.SubWeb, service.WebListenAddress))
	fmt.Printf("  URL prefix        = %q\n", srvm.GetConfig(service.SubWeb, service.WebURLPrefix))
	fmt.Println("Auth")
	fmt.Printf("  Read-only mode    = %v\n", srvm.GetConfig(service.SubAuth, service.AuthReadonly))
	if startup.WithAuth() {
		fmt.Printf("  Owner             = %v\n", startup.Owner())
		fmt.Printf("  Secure cookie     = %v\n", startup.SecureCookie())
		fmt.Printf("  Persistent cookie = %v\n", startup.PersistentCookie())
		htmlLifetime, apiLifetime := startup.TokenLifetime()
		fmt.Printf("  HTML lifetime     = %v\n", htmlLifetime)
		fmt.Printf("  API lifetime      = %v\n", apiLifetime)
	}

	return 0, nil
}
