//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// +build linux windows

// Package notifydir manages the notified directory part of a dirstore.
package notifydir

import (
	"log"
	"time"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/change"
	"zettelstore.de/z/place/dirplace/directory"
)

// ping sends every tick a signal to reload the directory list
func ping(tick chan<- struct{}, rescanTime time.Duration, done <-chan struct{}) {
	ticker := time.NewTicker(rescanTime)
	defer close(tick)
	for {
		select {
		case _, ok := <-ticker.C:
			if !ok {
				return
			}
			tick <- struct{}{}
		case _, ok := <-done:
			if !ok {
				ticker.Stop()
				return
			}
		}
	}
}

func newEntry(ev *fileEvent) *directory.Entry {
	de := new(directory.Entry)
	de.Zid = ev.zid
	updateEntry(de, ev)
	return de
}

func updateEntry(de *directory.Entry, ev *fileEvent) {
	if ev.ext == "meta" {
		de.MetaSpec = directory.MetaSpecFile
		de.MetaPath = ev.path
		return
	}
	if de.ContentExt != "" && de.ContentExt != ev.ext {
		de.Duplicates = true
		return
	}
	if de.MetaSpec != directory.MetaSpecFile {
		if ev.ext == "zettel" {
			de.MetaSpec = directory.MetaSpecHeader
		} else {
			de.MetaSpec = directory.MetaSpecNone
		}
	}
	de.ContentPath = ev.path
	de.ContentExt = ev.ext
}

type dirMap map[id.Zid]*directory.Entry

func dirMapUpdate(dm dirMap, ev *fileEvent) {
	de := dm[ev.zid]
	if de == nil {
		dm[ev.zid] = newEntry(ev)
		return
	}
	updateEntry(de, ev)
}

func deleteFromMap(dm dirMap, ev *fileEvent) {
	if ev.ext == "meta" {
		if entry, ok := dm[ev.zid]; ok {
			if entry.MetaSpec == directory.MetaSpecFile {
				entry.MetaSpec = directory.MetaSpecNone
				return
			}
		}
	}
	delete(dm, ev.zid)
}

// directoryService is the main service.
func (srv *notifyService) directoryService(events <-chan *fileEvent, ready chan<- int) {
	curMap := make(dirMap)
	var newMap dirMap
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return
			}
			switch ev.status {
			case fileStatusReloadStart:
				newMap = make(dirMap)
			case fileStatusReloadEnd:
				curMap = newMap
				newMap = nil
				if ready != nil {
					ready <- len(curMap)
					close(ready)
					ready = nil
				}
				srv.notifyChange(change.OnReload, id.Invalid)
			case fileStatusError:
				log.Println("DIRPLACE", "ERROR", ev.err)
			case fileStatusUpdate:
				srv.processFileUpdateEvent(ev, curMap, newMap)
			case fileStatusDelete:
				srv.processFileDeleteEvent(ev, curMap, newMap)
			}
		case cmd, ok := <-srv.cmds:
			if ok {
				cmd.run(curMap)
			}
		}
	}
}

func (srv *notifyService) processFileUpdateEvent(ev *fileEvent, curMap, newMap dirMap) {
	if newMap != nil {
		dirMapUpdate(newMap, ev)
	} else {
		dirMapUpdate(curMap, ev)
		srv.notifyChange(change.OnUpdate, ev.zid)
	}
}

func (srv *notifyService) processFileDeleteEvent(ev *fileEvent, curMap, newMap dirMap) {
	if newMap != nil {
		deleteFromMap(newMap, ev)
	} else {
		deleteFromMap(curMap, ev)
		srv.notifyChange(change.OnDelete, ev.zid)
	}
}

type dirCmd interface {
	run(m dirMap)
}

type cmdNumEntries struct {
	result chan<- resNumEntries
}
type resNumEntries = int

func (cmd *cmdNumEntries) run(m dirMap) {
	cmd.result <- len(m)
}

type cmdGetEntries struct {
	result chan<- resGetEntries
}
type resGetEntries []*directory.Entry

func (cmd *cmdGetEntries) run(m dirMap) {
	res := make([]*directory.Entry, len(m))
	i := 0
	for _, de := range m {
		entry := *de
		res[i] = &entry
		i++
	}
	cmd.result <- res
}

type cmdGetEntry struct {
	zid    id.Zid
	result chan<- resGetEntry
}
type resGetEntry = *directory.Entry

func (cmd *cmdGetEntry) run(m dirMap) {
	entry := m[cmd.zid]
	if entry == nil {
		cmd.result <- &directory.Entry{Zid: id.Invalid}
	} else {
		result := *entry
		cmd.result <- &result
	}
}

type cmdNewEntry struct {
	result chan<- resNewEntry
}
type resNewEntry = *directory.Entry

func (cmd *cmdNewEntry) run(m dirMap) {
	zid := id.New(false)
	if _, ok := m[zid]; !ok {
		entry := &directory.Entry{Zid: zid}
		m[zid] = entry
		result := *entry
		cmd.result <- &result
		return
	}
	for {
		zid = id.New(true)
		if _, ok := m[zid]; !ok {
			entry := &directory.Entry{Zid: zid}
			m[zid] = entry
			result := *entry
			cmd.result <- &result
			return
		}
		// TODO: do not wait here, but in a non-blocking goroutine.
		time.Sleep(100 * time.Millisecond)
	}
}

type cmdUpdateEntry struct {
	entry  *directory.Entry
	result chan<- struct{}
}

func (cmd *cmdUpdateEntry) run(m dirMap) {
	entry := *cmd.entry
	m[entry.Zid] = &entry
	cmd.result <- struct{}{}
}

type cmdRenameEntry struct {
	curEntry *directory.Entry
	newEntry *directory.Entry
	result   chan<- resRenameEntry
}

type resRenameEntry = error

func (cmd *cmdRenameEntry) run(m dirMap) {
	newEntry := *cmd.newEntry
	newZid := newEntry.Zid
	if _, found := m[newZid]; found {
		cmd.result <- &place.ErrInvalidID{Zid: newZid}
		return
	}
	delete(m, cmd.curEntry.Zid)
	m[newZid] = &newEntry
	cmd.result <- nil
}

type cmdDeleteEntry struct {
	zid    id.Zid
	result chan<- struct{}
}

func (cmd *cmdDeleteEntry) run(m dirMap) {
	delete(m, cmd.zid)
	cmd.result <- struct{}{}
}
