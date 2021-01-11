//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package startup provides functions to retrieve startup configuration data.
package startup

import (
	"os"
	"runtime"
)

// Version describes all elements of a software version.
type Version struct {
	Prog      string // Name of the software
	Build     string // Representation of build process
	Hostname  string // Host name a reported by the kernel
	GoVersion string // Version of go
	Os        string // GOOS
	Arch      string // GOARCH
	// More to come
}

var version Version

// SetupVersion initializes the version data.
func SetupVersion(progName, buildVersion string) {
	version.Prog = progName
	if buildVersion == "" {
		version.Build = "unknown"
	} else {
		version.Build = buildVersion
	}
	if hn, err := os.Hostname(); err == nil {
		version.Hostname = hn
	} else {
		version.Hostname = "*unknown host*"
	}
	version.GoVersion = runtime.Version()
	version.Os = runtime.GOOS
	version.Arch = runtime.GOARCH
}

// GetVersion returns the current software version data.
func GetVersion() Version { return version }
