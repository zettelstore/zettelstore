//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package directory manages the directory part of a directory place.
package directory

import (
	"log"
	"time"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/place"
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

func newEntry(ev *fileEvent) *Entry {
	de := new(Entry)
	de.Zid = ev.zid
	updateEntry(de, ev)
	return de
}

func updateEntry(de *Entry, ev *fileEvent) {
	if ev.ext == "meta" {
		de.MetaSpec = MetaSpecFile
		de.MetaPath = ev.path
		return
	}
	if de.ContentExt != "" && de.ContentExt != ev.ext {
		de.Duplicates = true
		return
	}
	if de.MetaSpec != MetaSpecFile {
		if ev.ext == "zettel" {
			de.MetaSpec = MetaSpecHeader
		} else {
			de.MetaSpec = MetaSpecNone
		}
	}
	de.ContentPath = ev.path
	de.ContentExt = ev.ext
}

type dirMap map[id.Zid]*Entry

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
			if entry.MetaSpec == MetaSpecFile {
				entry.MetaSpec = MetaSpecNone
				return
			}
		}
	}
	delete(dm, ev.zid)
}

// directoryService is the main service.
func (srv *Service) directoryService(events <-chan *fileEvent, ready chan<- int) {
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
				srv.notifyChange(place.OnReload, id.Invalid)
			case fileStatusError:
				log.Println("DIRPLACE", "ERROR", ev.err)
			case fileStatusUpdate:
				if newMap != nil {
					dirMapUpdate(newMap, ev)
				} else {
					dirMapUpdate(curMap, ev)
					srv.notifyChange(place.OnUpdate, ev.zid)
				}
			case fileStatusDelete:
				if newMap != nil {
					deleteFromMap(newMap, ev)
				} else {
					deleteFromMap(curMap, ev)
					srv.notifyChange(place.OnDelete, ev.zid)
				}
			}
		case cmd, ok := <-srv.cmds:
			if ok {
				cmd.run(curMap)
			}
		}
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
type resGetEntries []Entry

func (cmd *cmdGetEntries) run(m dirMap) {
	res := make([]Entry, 0, len(m))
	for _, de := range m {
		res = append(res, *de)
	}
	cmd.result <- res
}

type cmdGetEntry struct {
	zid    id.Zid
	result chan<- resGetEntry
}
type resGetEntry = Entry

func (cmd *cmdGetEntry) run(m dirMap) {
	entry := m[cmd.zid]
	if entry == nil {
		cmd.result <- Entry{Zid: id.Invalid}
	} else {
		cmd.result <- *entry
	}
}

type cmdNewEntry struct {
	result chan<- resNewEntry
}
type resNewEntry = Entry

func (cmd *cmdNewEntry) run(m dirMap) {
	zid := id.New(false)
	if _, ok := m[zid]; !ok {
		entry := &Entry{Zid: zid, MetaSpec: MetaSpecUnknown}
		m[zid] = entry
		cmd.result <- *entry
		return
	}
	for {
		zid = id.New(true)
		if _, ok := m[zid]; !ok {
			entry := &Entry{Zid: zid, MetaSpec: MetaSpecUnknown}
			m[zid] = entry
			cmd.result <- *entry
			return
		}
		// TODO: do not wait here, but in a non-blocking goroutine.
		time.Sleep(100 * time.Millisecond)
	}
}

type cmdUpdateEntry struct {
	entry  *Entry
	result chan<- struct{}
}

func (cmd *cmdUpdateEntry) run(m dirMap) {
	entry := *cmd.entry
	m[entry.Zid] = &entry
	cmd.result <- struct{}{}
}

type cmdRenameEntry struct {
	curEntry *Entry
	newEntry *Entry
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
