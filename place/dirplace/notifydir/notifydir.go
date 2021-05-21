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

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/dirplace/directory"
)

// notifyService specifies a directory scan service.
type notifyService struct {
	dirPath    string
	rescanTime time.Duration
	done       chan struct{}
	cmds       chan dirCmd
	infos      chan<- place.UpdateInfo
}

// NewService creates a new directory service.
func NewService(directoryPath string, rescanTime time.Duration, chci chan<- place.UpdateInfo) directory.Service {
	srv := &notifyService{
		dirPath:    directoryPath,
		rescanTime: rescanTime,
		cmds:       make(chan dirCmd),
		infos:      chci,
	}
	return srv
}

// Start makes the directory service operational.
func (srv *notifyService) Start() error {
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
	return nil
}

// Stop stops the directory service.
func (srv *notifyService) Stop() error {
	close(srv.done)
	srv.done = nil
	return nil
}

func (srv *notifyService) notifyChange(reason place.UpdateReason, zid id.Zid) {
	if chci := srv.infos; chci != nil {
		chci <- place.UpdateInfo{Reason: reason, Zid: zid}
	}
}

// NumEntries returns the number of managed zettel.
func (srv *notifyService) NumEntries() (int, error) {
	resChan := make(chan resNumEntries)
	srv.cmds <- &cmdNumEntries{resChan}
	return <-resChan, nil
}

// GetEntries returns an unsorted list of all current directory entries.
func (srv *notifyService) GetEntries() ([]*directory.Entry, error) {
	resChan := make(chan resGetEntries)
	srv.cmds <- &cmdGetEntries{resChan}
	return <-resChan, nil
}

// GetEntry returns the entry with the specified zettel id. If there is no such
// zettel id, an empty entry is returned.
func (srv *notifyService) GetEntry(zid id.Zid) (*directory.Entry, error) {
	resChan := make(chan resGetEntry)
	srv.cmds <- &cmdGetEntry{zid, resChan}
	return <-resChan, nil
}

// GetNew returns an entry with a new zettel id.
func (srv *notifyService) GetNew() (*directory.Entry, error) {
	resChan := make(chan resNewEntry)
	srv.cmds <- &cmdNewEntry{resChan}
	result := <-resChan
	return result.entry, result.err
}

// UpdateEntry notifies the directory of an updated entry.
func (srv *notifyService) UpdateEntry(entry *directory.Entry) error {
	resChan := make(chan struct{})
	srv.cmds <- &cmdUpdateEntry{entry, resChan}
	<-resChan
	return nil
}

// RenameEntry notifies the directory of an renamed entry.
func (srv *notifyService) RenameEntry(curEntry, newEntry *directory.Entry) error {
	resChan := make(chan resRenameEntry)
	srv.cmds <- &cmdRenameEntry{curEntry, newEntry, resChan}
	return <-resChan
}

// DeleteEntry removes a zettel id from the directory of entries.
func (srv *notifyService) DeleteEntry(zid id.Zid) error {
	resChan := make(chan struct{})
	srv.cmds <- &cmdDeleteEntry{zid, resChan}
	<-resChan
	return nil
}
