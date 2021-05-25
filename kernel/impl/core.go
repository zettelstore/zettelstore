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
	"fmt"
	"net"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"

	"zettelstore.de/z/kernel"
	"zettelstore.de/z/strfun"
)

type coreService struct {
	srvConfig
	started bool

	mxRecover  sync.RWMutex
	mapRecover map[string]recoverInfo
}
type recoverInfo struct {
	count uint64
	ts    time.Time
	info  interface{}
	stack []byte
}

func (cs *coreService) Initialize() {
	cs.mapRecover = make(map[string]recoverInfo)
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
	cs.mxRecover.RLock()
	defer cs.mxRecover.RUnlock()
	names := make([]string, 0, len(cs.mapRecover))
	for n := range cs.mapRecover {
		names = append(names, n)
	}
	sort.Strings(names)
	result := make([]kernel.KeyValue, 0, 3*len(names))
	for _, n := range names {
		ri := cs.mapRecover[n]
		result = append(
			result,
			kernel.KeyValue{
				Key:   fmt.Sprintf("Recover %q / Count", n),
				Value: fmt.Sprintf("%d", ri.count),
			},
			kernel.KeyValue{
				Key:   fmt.Sprintf("Recover %q / Last ", n),
				Value: fmt.Sprintf("%v", ri.ts),
			},
			kernel.KeyValue{
				Key:   fmt.Sprintf("Recover %q / Info ", n),
				Value: fmt.Sprintf("%v", ri.info),
			},
		)
	}
	return result
}

func (cs *coreService) RecoverLines(name string) []string {
	cs.mxRecover.RLock()
	ri := cs.mapRecover[name]
	cs.mxRecover.RUnlock()
	if ri.stack == nil {
		return nil
	}
	return append(
		[]string{
			fmt.Sprintf("Count: %d", ri.count),
			fmt.Sprintf("Time: %v", ri.ts),
			fmt.Sprintf("Reason: %v", ri.info),
		},
		strfun.SplitLines(string(ri.stack))...,
	)
}

func (cs *coreService) updateRecoverInfo(name string, recoverInfo interface{}, stack []byte) {
	cs.mxRecover.Lock()
	ri := cs.mapRecover[name]
	ri.count++
	ri.ts = time.Now()
	ri.info = recoverInfo
	ri.stack = stack
	cs.mapRecover[name] = ri
	cs.mxRecover.Unlock()
}
