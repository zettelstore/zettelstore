//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package directory

import (
	"path/filepath"
	"regexp"
	"sync"

	"zettelstore.de/z/box"
	"zettelstore.de/z/box/notify"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/kernel"
)

// dirService specifies a directory service for file based zettel.
type dirService struct {
	dirPath  string
	notifier notify.Notifier
	mx       sync.RWMutex
	entries  entrySet
}

type entrySet map[id.Zid]*Entry

// NewService creates a new directory service.
func NewService(directoryPath string, notifier notify.Notifier) Service {
	return &dirService{
		dirPath:  directoryPath,
		notifier: notifier,
	}
}

func (ds *dirService) Start() {
	go ds.updateEvents()
}

func (ds *dirService) Refresh() {
	ds.notifier.Refresh()
}

func (ds *dirService) Stop() {
	ds.notifier.Close()
}

func (ds *dirService) NumEntries() int {
	ds.mx.RLock()
	defer ds.mx.RUnlock()
	return len(ds.entries)
}

func (ds *dirService) GetEntries() []*Entry {
	ds.mx.RLock()
	defer ds.mx.RUnlock()
	result := make([]*Entry, 0, len(ds.entries))
	for _, entry := range ds.entries {
		copiedEntry := *entry
		result = append(result, &copiedEntry)
	}
	return result
}

func (ds *dirService) GetEntry(zid id.Zid) *Entry {
	ds.mx.RLock()
	defer ds.mx.RUnlock()
	foundEntry := ds.entries[zid]
	if foundEntry == nil {
		return nil
	}
	result := *foundEntry
	return &result
}

func (ds *dirService) GetNew() (id.Zid, error) {
	ds.mx.RLock()
	defer ds.mx.RUnlock()
	zid, err := box.GetNewZid(func(zid id.Zid) (bool, error) {
		_, found := ds.entries[zid]
		return !found, nil
	})
	if err != nil {
		return id.Invalid, err
	}
	ds.entries[zid] = &Entry{Zid: zid}
	return zid, nil
}

func (ds *dirService) UpdateEntry(updatedEntry *Entry) {
	entry := *updatedEntry
	ds.mx.Lock()
	ds.entries[entry.Zid] = &entry
	ds.mx.Unlock()
}

func (ds *dirService) RenameEntry(oldEntry, newEntry *Entry) error {
	ds.mx.Lock()
	defer ds.mx.Unlock()
	if _, found := ds.entries[newEntry.Zid]; found {
		return &box.ErrInvalidID{Zid: newEntry.Zid}
	}
	delete(ds.entries, oldEntry.Zid)
	entry := *newEntry
	ds.entries[entry.Zid] = &entry
	return nil
}

func (ds *dirService) DeleteEntry(zid id.Zid) {
	ds.mx.Lock()
	delete(ds.entries, zid)
	ds.mx.Unlock()
}

func (ds *dirService) updateEvents() {
	var newEntries entrySet
	for ev := range ds.notifier.Events() {
		switch ev.Op {
		case notify.Error:
			newEntries = nil
			kernel.Main.Log("ERROR", ev.Err)
		case notify.Make:
			newEntries = make(entrySet)
		case notify.List:
			if ev.Name == "" {
				ds.mx.Lock()
				ds.entries = newEntries
				newEntries = nil
				ds.mx.Unlock()
				continue
			}
			if newEntries != nil {
				ds.updateEntry(newEntries, ev.Name)
			}
		case notify.Destroy:
			ds.mx.Lock()
			ds.entries = nil
			ds.mx.Unlock()
		case notify.Update:
			ds.mx.Lock()
			ds.updateEntry(ds.entries, ev.Name)
			ds.mx.Unlock()
		case notify.Delete:
			ds.mx.Lock()
			deleteEntry(ds.entries, ev.Name)
			ds.mx.Unlock()
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

func fetchEntry(entries entrySet, zid id.Zid) *Entry {
	if entry, found := entries[zid]; found {
		return entry
	}
	entry := &Entry{Zid: zid}
	entries[zid] = entry
	return entry
}

func (ds *dirService) updateEntry(entries entrySet, name string) {
	if entries == nil {
		return
	}
	zid, ext := seekZidExt(name)
	if zid == id.Invalid {
		return
	}
	entry := fetchEntry(entries, zid)
	path := filepath.Join(ds.dirPath, name)
	// 	enhanceEntry(entry, path, ext)
	// }

	// func enhanceEntry(entry *Entry, path, ext string) {
	if ext == "meta" {
		entry.MetaSpec = MetaSpecFile
		entry.MetaPath = path
		return
	}
	if entry.ContentExt != "" && entry.ContentExt != ext {
		entry.Duplicates = true
		return
	}
	if entry.MetaSpec != MetaSpecFile {
		if ext == "zettel" {
			entry.MetaSpec = MetaSpecHeader
		} else {
			entry.MetaSpec = MetaSpecNone
		}
	}
	entry.ContentPath = path
	entry.ContentExt = ext
}

func deleteEntry(entries entrySet, name string) {
	if entries == nil {
		return
	}
	zid, ext := seekZidExt(name)
	if zid == id.Invalid {
		return
	}
	if ext == "meta" {
		if entry, found := entries[zid]; found {
			if entry.MetaSpec == MetaSpecFile {
				entry.MetaSpec = MetaSpecNone
				return
			}
		}
	}
	delete(entries, zid)
}
