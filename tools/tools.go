//-----------------------------------------------------------------------------
// Copyright (c) 2023-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2023-present Detlef Stern
//-----------------------------------------------------------------------------

package main

import (
	"fmt"
	"os"
)

func cmdTools() error {
	tools := []struct{ name, pack string }{
		{"shadow", "golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow@latest"},
		{"unparam", "mvdan.cc/unparam@latest"},
		{"staticcheck", "honnef.co/go/tools/cmd/staticcheck@latest"},
		{"govulncheck", "golang.org/x/vuln/cmd/govulncheck@latest"},
	}
	for _, tool := range tools {
		err := doGoInstall(tool.pack)
		if err != nil {
			return err
		}
	}
	return nil
}
func doGoInstall(pack string) error {
	out, err := executeCommand(nil, "go", "install", pack)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to install package", pack)
		if len(out) > 0 {
			fmt.Fprintln(os.Stderr, out)
		}
	}
	return err
}
