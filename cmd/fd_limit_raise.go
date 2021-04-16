//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// +build darwin

package cmd

import (
	"log"
	"syscall"
)

const minFiles = 1048576

func raiseFdLimit() error {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return err
	}
	if rLimit.Cur >= minFiles {
		return nil
	}
	rLimit.Cur = minFiles
	if rLimit.Cur > rLimit.Max {
		rLimit.Cur = rLimit.Max
	}
	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return err
	}
	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return err
	}
	if rLimit.Cur < minFiles {
		log.Printf("Make sure you have no more than %d files in all your places if you enabled notification\n", rLimit.Cur)
	}
	return nil
}
