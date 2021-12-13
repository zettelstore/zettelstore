//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package impl

import (
	"bufio"
	"net"
)

func startLineServer(kern *myKernel, listenAddr string) error {
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		kern.logger.Fatal().Err(err).Msg("Unable to start Line Command Server")
		return err
	}
	kern.logger.Mandatory().Str("listen", listenAddr).Msg("Start Line Command Server")
	go func() { lineServer(ln, kern) }()
	return nil
}

func lineServer(ln net.Listener, kern *myKernel) {
	// Something may panic. Ensure a running line service.
	defer func() {
		if r := recover(); r != nil {
			kern.doLogRecover("Line", r)
			go lineServer(ln, kern)
		}
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
			kern.logger.Error().Err(err).Msg("Unable to accept connection")
			break
		}
		go handleLineConnection(conn, kern)
	}
	ln.Close()
}

func handleLineConnection(conn net.Conn, kern *myKernel) {
	// Something may panic. Ensure a running connection.
	defer func() {
		if r := recover(); r != nil {
			kern.doLogRecover("LineConn", r)
			go handleLineConnection(conn, kern)
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
