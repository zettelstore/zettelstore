//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package notifydir manages the notified directory part of a dirstore.
package notifydir

import (
	"time"

	"zettelstore.de/z/box"
	"zettelstore.de/z/box/dirbox/directory"
	"zettelstore.de/z/domain/id"
)

// notifyService specifies a directory scan service.
type notifyService struct {
	dirPath    string
	rescanTime time.Duration
	done       chan struct{}
	cmds       chan dirCmd
	infos      chan<- box.UpdateInfo
}

// NewService creates a new directory service.
func NewService(directoryPath string, rescanTime time.Duration, chci chan<- box.UpdateInfo) directory.Service {
	srv := &notifyService{
		dirPath:    directoryPath,
		rescanTime: rescanTime,
		cmds:       make(chan dirCmd),
		infos:      chci,
	}
	return srv
}

// Start makes the directory service operational.
func (srv *notifyService) Start() {
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
func (srv *notifyService) Stop() {
	close(srv.done)
	srv.done = nil
}

func (*notifyService) Refresh() {}

func (srv *notifyService) notifyChange(reason box.UpdateReason, zid id.Zid) {
	if chci := srv.infos; chci != nil {
		chci <- box.UpdateInfo{Reason: reason, Zid: zid}
	}
}

// NumEntries returns the number of managed zettel.
func (srv *notifyService) NumEntries() int {
	resChan := make(chan resNumEntries)
	srv.cmds <- &cmdNumEntries{resChan}
	return <-resChan
}

// GetEntries returns an unsorted list of all current directory entries.
func (srv *notifyService) GetEntries() []*directory.Entry {
	resChan := make(chan resGetEntries)
	srv.cmds <- &cmdGetEntries{resChan}
	return <-resChan
}

// GetEntry returns the entry with the specified zettel id. If there is no such
// zettel id, an empty entry is returned.
func (srv *notifyService) GetEntry(zid id.Zid) *directory.Entry {
	resChan := make(chan resGetEntry)
	srv.cmds <- &cmdGetEntry{zid, resChan}
	return <-resChan
}

// GetNew returns an entry with a new zettel id.
func (srv *notifyService) GetNew() (id.Zid, error) {
	resChan := make(chan resNewEntry)
	srv.cmds <- &cmdNewEntry{resChan}
	result := <-resChan
	return result.zid, result.err
}

// UpdateEntry notifies the directory of an updated entry.
func (srv *notifyService) UpdateEntry(entry *directory.Entry) {
	resChan := make(chan struct{})
	srv.cmds <- &cmdUpdateEntry{entry, resChan}
	<-resChan
}

// RenameEntry notifies the directory of an renamed entry.
func (srv *notifyService) RenameEntry(curEntry, newEntry *directory.Entry) error {
	resChan := make(chan resRenameEntry)
	srv.cmds <- &cmdRenameEntry{curEntry, newEntry, resChan}
	return <-resChan
}

// DeleteEntry removes a zettel id from the directory of entries.
func (srv *notifyService) DeleteEntry(zid id.Zid) {
	resChan := make(chan struct{})
	srv.cmds <- &cmdDeleteEntry{zid, resChan}
	<-resChan
}
