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
	"log"
	"os"
	"strings"

	"zettelstore.de/z/config/startup"
	"zettelstore.de/z/web/server"
)

func flgSimpleRun(fs *flag.FlagSet) {
	fs.String("d", "", "zettel directory")
}

func runSimpleFunc(*flag.FlagSet) (int, error) {
	listenAddr := startup.ListenAddress()
	readonlyMode := startup.IsReadOnlyMode()
	logBeforeRun(listenAddr, readonlyMode)
	if idx := strings.LastIndexByte(listenAddr, ':'); idx >= 0 {
		log.Println()
		log.Println("--------------------------")
		log.Printf("Open your browser and enter the following URL:")
		log.Println()
		log.Printf("    http://localhost%v", listenAddr[idx:])
	}

	handler := setupRouting(startup.PlaceManager(), readonlyMode)
	srv := server.New(listenAddr, handler)
	if err := srv.Run(); err != nil {
		return 1, err
	}
	return 0, nil
}

// runSimple is called, when the user just starts the software via a double click
// or via a simple call ``./zettelstore`` on the command line.
func runSimple() {
	dir := "./zettel"
	if err := os.MkdirAll(dir, 0750); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create zettel directory %q (%s)\n", dir, err)
		os.Exit(1)
	}
	executeCommand("run-simple", "-d", dir)
}
