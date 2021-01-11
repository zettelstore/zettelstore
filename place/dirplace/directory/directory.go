//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package directory manages the directory part of a dirstore.
package directory

import (
	"sync"
	"time"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/place"
)

// Service specifies a directory scan service.
type Service struct {
	dirPath     string
	rescanTime  time.Duration
	done        chan struct{}
	cmds        chan dirCmd
	changeFuncs []place.ObserverFunc
	mxFuncs     sync.RWMutex
}

// NewService creates a new directory service.
func NewService(directoryPath string, rescanTime time.Duration) *Service {
	srv := &Service{
		dirPath:    directoryPath,
		rescanTime: rescanTime,
		cmds:       make(chan dirCmd),
	}
	return srv
}

// Start makes the directory service operational.
func (srv *Service) Start() {
	tick := make(chan struct{})
	rawEvents := make(chan *fileEvent)
	events := make(chan *fileEvent)

	ready := make(chan int)
	go srv.directoryService(events, ready)
	go collectEvents(events, rawEvents)
	go watchDirectory(srv.dirPath, rawEvents, tick)

	if srv.done != nil {
		panic("src.done already set")
	}
	srv.done = make(chan struct{})
	go ping(tick, srv.rescanTime, srv.done)
	<-ready
}

// Stop stops the directory service.
func (srv *Service) Stop() {
	close(srv.done)
	srv.done = nil
}

// Subscribe to invalidation events.
func (srv *Service) Subscribe(changeFunc place.ObserverFunc) {
	srv.mxFuncs.Lock()
	if changeFunc != nil {
		srv.changeFuncs = append(srv.changeFuncs, changeFunc)
	}
	srv.mxFuncs.Unlock()
}

func (srv *Service) notifyChange(reason place.ChangeReason, zid id.Zid) {
	srv.mxFuncs.RLock()
	changeFuncs := srv.changeFuncs
	srv.mxFuncs.RUnlock()
	for _, changeF := range changeFuncs {
		changeF(reason, zid)
	}
}

// NumEntries returns the number of managed zettel.
func (srv *Service) NumEntries() int {
	resChan := make(chan resNumEntries)
	srv.cmds <- &cmdNumEntries{resChan}
	return <-resChan
}

// GetEntries returns an unsorted list of all current directory entries.
func (srv *Service) GetEntries() []Entry {
	resChan := make(chan resGetEntries)
	srv.cmds <- &cmdGetEntries{resChan}
	return <-resChan
}

// GetEntry returns the entry with the specified zettel id. If there is no such
// zettel id, an empty entry is returned.
func (srv *Service) GetEntry(zid id.Zid) Entry {
	resChan := make(chan resGetEntry)
	srv.cmds <- &cmdGetEntry{zid, resChan}
	return <-resChan
}

// GetNew returns an entry with a new zettel id.
func (srv *Service) GetNew() Entry {
	resChan := make(chan resNewEntry)
	srv.cmds <- &cmdNewEntry{resChan}
	return <-resChan
}

// UpdateEntry notifies the directory of an updated entry.
func (srv *Service) UpdateEntry(entry *Entry) {
	resChan := make(chan struct{})
	srv.cmds <- &cmdUpdateEntry{entry, resChan}
	<-resChan
}

// RenameEntry notifies the directory of an renamed entry.
func (srv *Service) RenameEntry(curEntry, newEntry *Entry) error {
	resChan := make(chan resRenameEntry)
	srv.cmds <- &cmdRenameEntry{curEntry, newEntry, resChan}
	return <-resChan
}

// DeleteEntry removes a zettel id from the directory of entries.
func (srv *Service) DeleteEntry(zid id.Zid) {
	resChan := make(chan struct{})
	srv.cmds <- &cmdDeleteEntry{zid, resChan}
	<-resChan
}
