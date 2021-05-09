//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package impl provides the main internal service implementation.
package impl

import (
	"os"
	"runtime"

	"zettelstore.de/z/service"
)

type coreSub struct {
	subConfig
}

func (ms *coreSub) Initialize() {
	ms.descr = descriptionMap{
		service.CoreGoArch:    {"Go processor architecture", nil, false},
		service.CoreGoOS:      {"Go Operating System", nil, false},
		service.CoreGoVersion: {"Go Version", nil, false},
		service.CoreHostname:  {"Host name", nil, false},
		service.CoreProgname:  {"Program name", nil, false},
		service.CoreVerbose:   {"Verbose output", parseBool, true},
		service.CoreVersion: {
			"Version",
			ms.noFrozen(func(val string) interface{} {
				if val == "" {
					return "unknown"
				}
				return val
			}),
			false,
		},
	}
	ms.next = interfaceMap{
		service.CoreGoArch:    runtime.GOARCH,
		service.CoreGoOS:      runtime.GOOS,
		service.CoreGoVersion: runtime.Version(),
		service.CoreHostname:  "*unknwon host*",
		service.CoreVerbose:   false,
	}
	if hn, err := os.Hostname(); err == nil {
		ms.next[service.CoreHostname] = hn
	}
}

func (ms *coreSub) Start(srv *myService) error {
	return nil
}
func (ms *coreSub) Stop(srv *myService) error {
	return nil
}
