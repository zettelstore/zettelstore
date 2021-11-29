//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package simpledir manages the directory part of a dirstore.
package simpledir

import (
	"path/filepath"
	"regexp"
	"sync"

	"zettelstore.de/z/box"
	"zettelstore.de/z/box/dirbox/directory"
	"zettelstore.de/z/box/notify"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/kernel"
)

// simpleService specifies a directory service without scanning.
type simpleService struct {
	dirPath  string
	notifier notify.Notifier
	mx       sync.Mutex
	entries  entrySet
}

type entrySet map[id.Zid]*directory.Entry

// NewService creates a new directory service.
func NewService(directoryPath string, notifier notify.Notifier) directory.Service {
	return &simpleService{
		dirPath:  directoryPath,
		notifier: notifier,
	}
}

func (ss *simpleService) Start() {
	go ss.updateEvents()
}

func (ss *simpleService) Refresh() {
	ss.notifier.Refresh()
}

func (ss *simpleService) Stop() {
	ss.notifier.Close()
}

func (ss *simpleService) NumEntries() int {
	ss.mx.Lock()
	defer ss.mx.Unlock()
	return len(ss.entries)
}

func (ss *simpleService) GetEntries() []*directory.Entry {
	ss.mx.Lock()
	defer ss.mx.Unlock()
	result := make([]*directory.Entry, 0, len(ss.entries))
	for _, entry := range ss.entries {
		copiedEntry := *entry
		result = append(result, &copiedEntry)
	}
	return result
}

func (ss *simpleService) GetEntry(zid id.Zid) *directory.Entry {
	ss.mx.Lock()
	defer ss.mx.Unlock()
	foundEntry := ss.entries[zid]
	if foundEntry == nil {
		return nil
	}
	result := *foundEntry
	return &result
}

func (ss *simpleService) GetNew() (id.Zid, error) {
	ss.mx.Lock()
	defer ss.mx.Unlock()
	zid, err := box.GetNewZid(func(zid id.Zid) (bool, error) {
		_, found := ss.entries[zid]
		return !found, nil
	})
	if err != nil {
		return id.Invalid, err
	}
	ss.entries[zid] = &directory.Entry{Zid: zid}
	return zid, nil
}

func (ss *simpleService) UpdateEntry(updatedEntry *directory.Entry) {
	ss.mx.Lock()
	defer ss.mx.Unlock()
	entry := *updatedEntry
	ss.entries[entry.Zid] = &entry
}

func (ss *simpleService) RenameEntry(oldEntry, newEntry *directory.Entry) error {
	ss.mx.Lock()
	defer ss.mx.Unlock()
	if _, found := ss.entries[newEntry.Zid]; found {
		return &box.ErrInvalidID{Zid: newEntry.Zid}
	}
	delete(ss.entries, oldEntry.Zid)
	entry := *newEntry
	ss.entries[entry.Zid] = &entry
	return nil
}

func (ss *simpleService) DeleteEntry(zid id.Zid) {
	ss.mx.Lock()
	defer ss.mx.Unlock()
	delete(ss.entries, zid)
}

func (ss *simpleService) updateEvents() {
	var newEntries entrySet
	for ev := range ss.notifier.Events() {
		switch ev.Op {
		case notify.Error:
			newEntries = nil
			kernel.Main.Log("ERROR", ev.Err)
		case notify.Make:
			newEntries = make(entrySet)
		case notify.List:
			if ev.Name == "" {
				ss.mx.Lock()
				ss.entries = newEntries
				newEntries = nil
				ss.mx.Unlock()
				continue
			}
			if newEntries != nil {
				ss.updateEntry(newEntries, ev.Name)
			}
		case notify.Destroy:
			ss.mx.Lock()
			ss.entries = nil
			ss.mx.Unlock()
		case notify.Update:
			ss.mx.Lock()
			ss.updateEntry(ss.entries, ev.Name)
			ss.mx.Unlock()
		case notify.Delete:
			ss.mx.Lock()
			ss.deleteEntry(ss.entries, ev.Name)
			ss.mx.Unlock()
		default:
			kernel.Main.Log("Unknown event", ev)
		}
	}
}

var validFileName = regexp.MustCompile(`^(\d{14}).*(\.(.+))$`)

func matchValidFileName(name string) []string {
	return validFileName.FindStringSubmatch(name)
}

func seekZidExt(name string) (id.Zid, string) {
	match := matchValidFileName(name)
	if len(match) == 0 {
		return id.Invalid, ""
	}
	zid, err := id.Parse(match[1])
	if err != nil {
		return id.Invalid, ""
	}
	return zid, match[3]
}

func fetchEntry(entries entrySet, zid id.Zid) *directory.Entry {
	if entry, found := entries[zid]; found {
		return entry
	}
	entry := &directory.Entry{Zid: zid}
	entries[zid] = entry
	return entry
}

func (ss *simpleService) updateEntry(entries entrySet, name string) {
	if entries == nil {
		return
	}
	zid, ext := seekZidExt(name)
	if zid == id.Invalid {
		return
	}
	entry := fetchEntry(entries, zid)
	path := filepath.Join(ss.dirPath, name)
	// 	enhanceEntry(entry, path, ext)
	// }

	// func enhanceEntry(entry *directory.Entry, path, ext string) {
	if ext == "meta" {
		entry.MetaSpec = directory.MetaSpecFile
		entry.MetaPath = path
		return
	}
	if entry.ContentExt != "" && entry.ContentExt != ext {
		entry.Duplicates = true
		return
	}
	if entry.MetaSpec != directory.MetaSpecFile {
		if ext == "zettel" {
			entry.MetaSpec = directory.MetaSpecHeader
		} else {
			entry.MetaSpec = directory.MetaSpecNone
		}
	}
	entry.ContentPath = path
	entry.ContentExt = ext
}

func (ss *simpleService) deleteEntry(entries entrySet, name string) {
	if entries == nil {
		return
	}
	zid, ext := seekZidExt(name)
	if zid == id.Invalid {
		return
	}
	if ext == "meta" {
		if entry, found := entries[zid]; found {
			if entry.MetaSpec == directory.MetaSpecFile {
				entry.MetaSpec = directory.MetaSpecNone
				return
			}
		}
	}
	delete(entries, zid)
}
