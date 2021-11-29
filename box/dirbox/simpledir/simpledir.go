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

func (ss *simpleService) updateEvents() {
	var newEntries entrySet
	for ev := range ss.notifier.Events() {
		switch ev.Op {
		case notify.Error:
			newEntries = nil
			panic(ev.Err)
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
				ss.addEntry(newEntries, ev.Name)
			}
		}
	}
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
		result = append(result, entry)
	}
	return result
}

func (ss *simpleService) addEntry(entries entrySet, name string) {
	match := matchValidFileName(name)
	if len(match) == 0 {
		return
	}
	zid, err2 := id.Parse(match[1])
	if err2 != nil {
		return
	}
	var entry *directory.Entry
	if e, ok := entries[zid]; ok {
		entry = e
	} else {
		entry = &directory.Entry{Zid: zid}
		entries[zid] = entry
	}
	updateEntry(entry, filepath.Join(ss.dirPath, name), match[3])
}

var validFileName = regexp.MustCompile(`^(\d{14}).*(\.(.+))$`)

func matchValidFileName(name string) []string {
	return validFileName.FindStringSubmatch(name)
}

func updateEntry(entry *directory.Entry, path, ext string) {
	if ext == "meta" {
		entry.MetaSpec = directory.MetaSpecFile
		entry.MetaPath = path
	} else if entry.ContentExt != "" && entry.ContentExt != ext {
		entry.Duplicates = true
	} else {
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
}

func (ss *simpleService) GetEntry(zid id.Zid) *directory.Entry {
	ss.mx.Lock()
	defer ss.mx.Unlock()
	return ss.entries[zid]
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
	return zid, nil
}

func (ss *simpleService) UpdateEntry(entry *directory.Entry) {
	ss.mx.Lock()
	defer ss.mx.Unlock()
	ss.entries[entry.Zid] = entry
}

func (ss *simpleService) RenameEntry(oldEntry, newEntry *directory.Entry) error {
	ss.mx.Lock()
	defer ss.mx.Unlock()
	if _, found := ss.entries[newEntry.Zid]; found {
		return &box.ErrInvalidID{Zid: newEntry.Zid}
	}
	if _, found := ss.entries[oldEntry.Zid]; found {
		delete(ss.entries, oldEntry.Zid)
		ss.entries[newEntry.Zid] = newEntry
	}
	return nil
}

func (ss *simpleService) DeleteEntry(zid id.Zid) {
	ss.mx.Lock()
	defer ss.mx.Unlock()
	delete(ss.entries, zid)
}
