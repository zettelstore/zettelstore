//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package notifydir

import (
	"time"

	"zettelstore.de/z/box"
	"zettelstore.de/z/box/dirbox/directory"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/kernel"
)

// ping sends every tick a signal to reload the directory list
func ping(tick chan<- struct{}, rescanTime time.Duration, done <-chan struct{}) {
	// Something may panic. Ensure a running service.
	defer func() {
		if r := recover(); r != nil {
			kernel.Main.LogRecover("Ping", r)
			go ping(tick, rescanTime, done)
		}
	}()

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
	// Something may panic. Ensure a running service.
	defer func() {
		if r := recover(); r != nil {
			kernel.Main.LogRecover("DirectoryService", r)
			go srv.directoryService(events, ready)
		}
	}()

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
				srv.notifyChange(box.OnReload, id.Invalid)
			case fileStatusError:
				srv.log.Warn().Err(ev.err).Msg("FSNotify")
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
		srv.notifyChange(box.OnUpdate, ev.zid)
	}
}

func (srv *notifyService) processFileDeleteEvent(ev *fileEvent, curMap, newMap dirMap) {
	if newMap != nil {
		deleteFromMap(newMap, ev)
	} else {
		deleteFromMap(curMap, ev)
		srv.notifyChange(box.OnDelete, ev.zid)
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
		cmd.result <- nil
	} else {
		result := *entry
		cmd.result <- &result
	}
}

type cmdNewEntry struct {
	result chan<- resNewEntry
}
type resNewEntry struct {
	zid id.Zid
	err error
}

func (cmd *cmdNewEntry) run(m dirMap) {
	zid, err := box.GetNewZid(func(zid id.Zid) (bool, error) {
		_, ok := m[zid]
		return !ok, nil
	})
	if err != nil {
		cmd.result <- resNewEntry{id.Invalid, err}
		return
	}
	m[zid] = &directory.Entry{Zid: zid}
	cmd.result <- resNewEntry{zid, nil}
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
		cmd.result <- &box.ErrInvalidID{Zid: newZid}
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
