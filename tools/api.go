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

package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
	"time"
)

type zsInfo struct {
	cmd          *exec.Cmd
	out          strings.Builder
	adminAddress string
}

func cmdTestAPI() error {
	var err error
	var info zsInfo
	needServer := !addressInUse(":23123")
	if needServer {
		err = startZettelstore(&info)
	}
	if err != nil {
		return err
	}
	err = checkGoTest("zettelstore.de/z/tests/client", "-base-url", "http://127.0.0.1:23123")
	if needServer {
		err1 := stopZettelstore(&info)
		if err == nil {
			err = err1
		}
	}
	return err
}

func startZettelstore(info *zsInfo) error {
	info.adminAddress = ":2323"
	name, arg := "go", []string{
		"run", "cmd/zettelstore/main.go", "run",
		"-c", "./testdata/testbox/19700101000000.zettel", "-a", info.adminAddress[1:]}
	logCommand("FORK", nil, name, arg)
	cmd := prepareCommand(envGoVCS, name, arg, &info.out)
	if !verbose {
		cmd.Stderr = nil
	}
	err := cmd.Start()
	time.Sleep(2 * time.Second)
	for i := 0; i < 100; i++ {
		time.Sleep(time.Millisecond * 100)
		if addressInUse(info.adminAddress) {
			info.cmd = cmd
			return err
		}
	}
	time.Sleep(4 * time.Second) // Wait for all zettel to be indexed.
	return errors.New("zettelstore did not start")
}

func stopZettelstore(i *zsInfo) error {
	conn, err := net.Dial("tcp", i.adminAddress)
	if err != nil {
		fmt.Println("Unable to stop Zettelstore")
		return err
	}
	io.WriteString(conn, "shutdown\n")
	conn.Close()
	err = i.cmd.Wait()
	return err
}

func addressInUse(address string) bool {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
