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

// Package main provides a command to execute unit tests.
package main

import (
	"flag"
	"fmt"
	"os"

	"zettelstore.de/z/tools"
)

var release bool

func main() {
	flag.BoolVar(&release, "r", false, "Release check")
	flag.BoolVar(&tools.Verbose, "v", false, "Verbose output")
	flag.Parse()

	if err := tools.Check(release); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
