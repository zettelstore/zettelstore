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

// Package main provides a command to clean / remove development artifacts.
package main

import (
	"flag"
	"fmt"
	"os"

	"zettelstore.de/z/tools"
)

func main() {
	flag.BoolVar(&tools.Verbose, "v", false, "Verbose output")
	flag.Parse()

	if err := cmdClean(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func cmdClean() error {
	for _, dir := range []string{"bin", "releases"} {
		err := os.RemoveAll(dir)
		if err != nil {
			return err
		}
	}
	out, err := tools.ExecuteCommand(nil, "go", "clean", "./...")
	if err != nil {
		return err
	}
	if len(out) > 0 {
		fmt.Println(out)
	}
	out, err = tools.ExecuteCommand(nil, "go", "clean", "-cache", "-modcache", "-testcache")
	if err != nil {
		return err
	}
	if len(out) > 0 {
		fmt.Println(out)
	}
	return nil
}
