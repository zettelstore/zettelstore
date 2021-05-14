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
	"bufio"
	"net"
	"os"
	"runtime"
	"strconv"

	"zettelstore.de/z/service"
)

type coreSub struct {
	subConfig
	started bool
}

func (cs *coreSub) Initialize() {
	cs.descr = descriptionMap{
		service.CoreGoArch:    {"Go processor architecture", nil, false},
		service.CoreGoOS:      {"Go Operating System", nil, false},
		service.CoreGoVersion: {"Go Version", nil, false},
		service.CoreHostname:  {"Host name", nil, false},
		service.CorePort: {
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
		service.CoreProgname: {"Program name", nil, false},
		service.CoreVerbose:  {"Verbose output", parseBool, true},
		service.CoreVersion: {
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
		service.CoreGoArch:    runtime.GOARCH,
		service.CoreGoOS:      runtime.GOOS,
		service.CoreGoVersion: runtime.Version(),
		service.CoreHostname:  "*unknwon host*",
		service.CorePort:      0,
		service.CoreVerbose:   false,
	}
	if hn, err := os.Hostname(); err == nil {
		cs.next[service.CoreHostname] = hn
	}
}

func (cs *coreSub) Start(srv *myService) error {
	port := cs.GetNextConfig(service.CorePort).(int)
	if port <= 0 {
		return nil
	}
	srvname := cs.GetNextConfig(service.CoreProgname).(string) + " Core Service"
	listenAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		srv.doLog("Unable to start", srvname, err)
		return err
	}
	srv.doLog("Start", srvname, "on", listenAddr)
	go func() { cs.serve(ln, srv) }()
	cs.started = true
	return nil
}
func (cs *coreSub) IsStarted() bool { return cs.started }
func (cs *coreSub) Stop(srv *myService) error {
	return nil
}

func (cs *coreSub) serve(ln net.Listener, srv *myService) {
	// Something may panic. Ensure a running line service.
	defer func() {
		if r := recover(); r != nil {
			srv.doLogRecover("Line", r)
			go cs.serve(ln, srv)
		}
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
			srv.doLog("Unable to accept connection:", err)
			break
		}
		go handleConnection(conn, srv)
	}
	ln.Close()
}

func handleConnection(conn net.Conn, srv *myService) {
	// Something may panic. Ensure a running connection.
	defer func() {
		if r := recover(); r != nil {
			srv.doLogRecover("LineConn", r)
			go handleConnection(conn, srv)
		}
	}()

	cmds := cmdSession{}
	cmds.initialize(conn, srv)
	s := bufio.NewScanner(conn)
	for s.Scan() {
		line := s.Text()
		if !cmds.executeLine(line) {
			break
		}
	}
	conn.Close()
}

func (cs *coreSub) GetStatistics() []service.KeyValue {
	return nil
}
