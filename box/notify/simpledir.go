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

import "path/filepath"

type simpleDirNotifier struct {
	events chan Event
	done   chan struct{}
	reload chan struct{}
	path   string
}

// NewSimpleDirNotifier creates a directory based notifier that will not receive
// any notifications from the operating system.
func NewSimpleDirNotifier(path string) (Notifier, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	sdn := &simpleDirNotifier{
		events: make(chan Event),
		done:   make(chan struct{}),
		reload: make(chan struct{}),
		path:   absPath,
	}
	go sdn.eventLoop()
	return sdn, nil
}

func (sdn *simpleDirNotifier) Events() <-chan Event {
	return sdn.events
}

func (sdn *simpleDirNotifier) Reload() {
	sdn.reload <- struct{}{}
}

func (sdn *simpleDirNotifier) eventLoop() {
	defer close(sdn.events)
	defer close(sdn.reload)
	if !listDirElements(sdn.path, sdn.events, sdn.done) {
		return
	}
	for {
		select {
		case <-sdn.done:
			return
		case <-sdn.reload:
			listDirElements(sdn.path, sdn.events, sdn.done)
		}
	}
}

func (sdn *simpleDirNotifier) Close() {
	close(sdn.done)
}
