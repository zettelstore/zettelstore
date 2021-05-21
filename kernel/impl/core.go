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
	"bufio"
	"net"
	"os"
	"runtime"
	"strconv"

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
			"Port of core service",
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
		kernel.CoreHostname:  "*unknwon host*",
		kernel.CorePort:      0,
		kernel.CoreVerbose:   false,
	}
	if hn, err := os.Hostname(); err == nil {
		cs.next[kernel.CoreHostname] = hn
	}
}

func (cs *coreService) Start(kern *myKernel) error {
	port := cs.GetNextConfig(kernel.CorePort).(int)
	if port <= 0 {
		return nil
	}
	listenAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		kern.doLog("Unable to start Core Service:", err)
		return err
	}
	kern.doLog("Start Core Service:", listenAddr)
	go func() { cs.serve(ln, kern) }()
	cs.started = true
	return nil
}
func (cs *coreService) IsStarted() bool { return cs.started }
func (cs *coreService) Stop(*myKernel) error {
	return nil
}

func (cs *coreService) serve(ln net.Listener, kern *myKernel) {
	// Something may panic. Ensure a running line service.
	defer func() {
		if r := recover(); r != nil {
			kern.doLogRecover("Line", r)
			go cs.serve(ln, kern)
		}
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
			kern.doLog("Unable to accept connection:", err)
			break
		}
		go handleConnection(conn, kern)
	}
	ln.Close()
}

func handleConnection(conn net.Conn, kern *myKernel) {
	// Something may panic. Ensure a running connection.
	defer func() {
		if r := recover(); r != nil {
			kern.doLogRecover("LineConn", r)
			go handleConnection(conn, kern)
		}
	}()

	cmds := cmdSession{}
	cmds.initialize(conn, kern)
	s := bufio.NewScanner(conn)
	for s.Scan() {
		line := s.Text()
		if !cmds.executeLine(line) {
			break
		}
	}
	conn.Close()
}

func (cs *coreService) GetStatistics() []kernel.KeyValue {
	return nil
}
