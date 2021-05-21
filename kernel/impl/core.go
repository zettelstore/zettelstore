//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package impl provides the kernel implementation.
package impl

import (
	"net"
	"os"
	"runtime"

	"zettelstore.de/z/kernel"
)

type coreService struct {
	srvConfig
	started bool
}

func (cs *coreService) Initialize() {
	cs.descr = descriptionMap{
		kernel.CoreGoArch:    {"Go processor architecture", nil, false},
		kernel.CoreGoOS:      {"Go Operating System", nil, false},
		kernel.CoreGoVersion: {"Go Version", nil, false},
		kernel.CoreHostname:  {"Host name", nil, false},
		kernel.CorePort: {
			"Port of command line server",
			cs.noFrozen(func(val string) interface{} {
				port, err := net.LookupPort("tcp", val)
				if err != nil {
					return nil
				}
				return port
			}),
			true,
		},
		kernel.CoreProgname: {"Program name", nil, false},
		kernel.CoreVerbose:  {"Verbose output", parseBool, true},
		kernel.CoreVersion: {
			"Version",
			cs.noFrozen(func(val string) interface{} {
				if val == "" {
					return "unknown"
				}
				return val
			}),
			false,
		},
	}
	cs.next = interfaceMap{
		kernel.CoreGoArch:    runtime.GOARCH,
		kernel.CoreGoOS:      runtime.GOOS,
		kernel.CoreGoVersion: runtime.Version(),
		kernel.CoreHostname:  "*unknown host*",
		kernel.CorePort:      0,
		kernel.CoreVerbose:   false,
	}
	if hn, err := os.Hostname(); err == nil {
		cs.next[kernel.CoreHostname] = hn
	}
}

func (cs *coreService) Start(kern *myKernel) error {
	cs.started = true
	return nil
}
func (cs *coreService) IsStarted() bool { return cs.started }
func (cs *coreService) Stop(*myKernel) error {
	cs.started = false
	return nil
}

func (cs *coreService) GetStatistics() []kernel.KeyValue {
	return nil
}
