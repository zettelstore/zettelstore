//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package main is the starting point for the zettelstore command.
package main

import (
	"zettelstore.de/z/cmd"
)

// Version variable. Will be filled by build process.
var buildVersion string = ""

func main() {
	cmd.Main("Zettelstore", buildVersion)
}
