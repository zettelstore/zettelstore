//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"zettelstore.de/z/kernel"
)

func flgSimpleRun(fs *flag.FlagSet) {
	fs.String("d", "", "zettel directory")
}

func runSimpleFunc(*flag.FlagSet) (int, error) {
	kern := kernel.Main
	logger := kern.GetKernelLogger()
	listenAddr := kern.GetConfig(kernel.WebService, kernel.WebListenAddress).(string)
	exitCode, err := doRun()
	if idx := strings.LastIndexByte(listenAddr, ':'); idx >= 0 {
		logger.Mandatory().Msg(strings.Repeat("--------------------", 3))
		logger.Mandatory().Msg("Open your browser and enter the following URL:")
		logger.Mandatory().Msg(fmt.Sprintf("    http://localhost%v", listenAddr[idx:]))
	}
	kern.WaitForShutdown()
	return exitCode, err
}

// runSimple is called, when the user just starts the software via a double click
// or via a simple call ``./zettelstore`` on the command line.
func runSimple() int {
	dir := "./zettel"
	if err := os.MkdirAll(dir, 0750); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create zettel directory %q (%s)\n", dir, err)
		os.Exit(1)
	}
	return executeCommand("run-simple", "-d", dir)
}
