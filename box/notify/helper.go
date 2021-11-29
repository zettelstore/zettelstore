//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package notify

import "os"

// listDirElements write all files within the directory path as events.
func listDirElements(dirPath string, events chan<- Event, done <-chan struct{}) bool {
	select {
	case events <- Event{Op: Make}:
	case <-done:
		return false
	}
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		select {
		case events <- Event{Op: Error, Err: err}:
		case <-done:
			return false
		}
	}
	for _, entry := range entries {
		if info, err1 := entry.Info(); err1 != nil || !info.Mode().IsRegular() {
			continue
		}
		select {
		case events <- Event{Op: List, Name: entry.Name()}:
		case <-done:
			return false
		}
	}

	select {
	case events <- Event{Op: List}:
	case <-done:
		return false
	}
	return true
}
