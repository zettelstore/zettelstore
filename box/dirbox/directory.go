//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package dirbox

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sync"

	"zettelstore.de/z/box"
	"zettelstore.de/z/box/notify"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/logger"
)

// dirMetaSpec defines all possibilities where meta data can be stored.
type dirMetaSpec uint8

// Constants for MetaSpec
const (
	_                 dirMetaSpec = iota
	dirMetaSpecNone               // no meta information
	dirMetaSpecFile               // meta information is in meta file
	dirMetaSpecHeader             // meta information is in header
)

// dirEntry stores everything for a directory entry.
type dirEntry struct {
	zid         id.Zid
	metaSpec    dirMetaSpec // location of meta information
	duplicates  bool        // multiple content files
	metaPath    string      // file path of meta information
	contentPath string      // file path of zettel content
	contentExt  string      // (normalized) file extension of zettel content
}

// isValid checks whether the entry is valid.
func (e *dirEntry) isValid() bool {
	return e != nil && e.zid.IsValid()
}

// dirService specifies a directory service for file based zettel.
type dirService struct {
	log      *logger.Logger
	dirPath  string
	notifier notify.Notifier
	infos    chan<- box.UpdateInfo
	mx       sync.RWMutex
	entries  entrySet
}

type entrySet map[id.Zid]*dirEntry

// newDirService creates a new directory service.
func newDirService(log *logger.Logger, directoryPath string, notifier notify.Notifier, chci chan<- box.UpdateInfo) *dirService {
	return &dirService{
		log:      log,
		dirPath:  directoryPath,
		notifier: notifier,
		infos:    chci,
	}
}

func (ds *dirService) startDirService() {
	go ds.updateEvents()
}

func (ds *dirService) refreshDirService() {
	ds.notifier.Refresh()
}

func (ds *dirService) stopDirService() {
	ds.notifier.Close()
}

func (ds *dirService) countDirEntries() int {
	ds.mx.RLock()
	defer ds.mx.RUnlock()
	return len(ds.entries)
}

func (ds *dirService) getDirEntries() []*dirEntry {
	ds.mx.RLock()
	defer ds.mx.RUnlock()
	result := make([]*dirEntry, 0, len(ds.entries))
	for _, entry := range ds.entries {
		copiedEntry := *entry
		result = append(result, &copiedEntry)
	}
	return result
}

func (ds *dirService) getDirEntry(zid id.Zid) *dirEntry {
	ds.mx.RLock()
	defer ds.mx.RUnlock()
	foundEntry := ds.entries[zid]
	if foundEntry == nil {
		return nil
	}
	result := *foundEntry
	return &result
}

func (ds *dirService) calcNewDirEntry() (id.Zid, error) {
	ds.mx.RLock()
	defer ds.mx.RUnlock()
	zid, err := box.GetNewZid(func(zid id.Zid) (bool, error) {
		_, found := ds.entries[zid]
		return !found, nil
	})
	if err != nil {
		return id.Invalid, err
	}
	ds.entries[zid] = &dirEntry{zid: zid}
	return zid, nil
}

func (ds *dirService) updateDirEntry(updatedEntry *dirEntry) {
	entry := *updatedEntry
	ds.mx.Lock()
	ds.entries[entry.zid] = &entry
	ds.mx.Unlock()
}

func (ds *dirService) renameDirEntry(oldEntry, newEntry *dirEntry) error {
	ds.mx.Lock()
	defer ds.mx.Unlock()
	if _, found := ds.entries[newEntry.zid]; found {
		return &box.ErrInvalidID{Zid: newEntry.zid}
	}
	delete(ds.entries, oldEntry.zid)
	entry := *newEntry
	ds.entries[entry.zid] = &entry
	return nil
}

func (ds *dirService) deleteDirEntry(zid id.Zid) {
	ds.mx.Lock()
	delete(ds.entries, zid)
	ds.mx.Unlock()
}

func (ds *dirService) updateEvents() {
	var newEntries entrySet
	for ev := range ds.notifier.Events() {
		ds.log.Trace().Str("op", ev.Op.String()).Str("name", ev.Name).Msg("notifyEvent")
		switch ev.Op {
		case notify.Error:
			newEntries = nil
			ds.log.Warn().Err(ev.Err).Msg("Notifier confused")
		case notify.Make:
			newEntries = make(entrySet)
		case notify.List:
			if ev.Name == "" {
				ds.mx.Lock()
				zids := make(id.Slice, 0, len(newEntries))
				for zid := range newEntries {
					zids = append(zids, zid)
				}
				ds.entries = newEntries
				newEntries = nil
				ds.mx.Unlock()
				for _, zid := range zids {
					ds.notifyChange(box.OnUpdate, zid)
				}
				ds.notifyChange(box.OnReload, id.Invalid)
			} else if newEntries != nil {
				ds.onUpdateFileEvent(newEntries, ev.Name)
			}
		case notify.Destroy:
			ds.mx.Lock()
			ds.entries = nil
			ds.mx.Unlock()
		case notify.Update:
			ds.mx.Lock()
			zid := ds.onUpdateFileEvent(ds.entries, ev.Name)
			ds.mx.Unlock()
			if zid != id.Invalid {
				ds.notifyChange(box.OnUpdate, zid)
			}
		case notify.Delete:
			ds.mx.Lock()
			ds.onDeleteFileEvent(ds.entries, ev.Name)
			ds.mx.Unlock()
		default:
			ds.log.Warn().Str("event", fmt.Sprintf("%v", ev)).Msg("Unknown event")
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

func fetchdirEntry(entries entrySet, zid id.Zid) *dirEntry {
	if entry, found := entries[zid]; found {
		return entry
	}
	entry := &dirEntry{zid: zid}
	entries[zid] = entry
	return entry
}

func (ds *dirService) onUpdateFileEvent(entries entrySet, name string) id.Zid {
	if entries == nil {
		return id.Invalid
	}
	zid, ext := seekZidExt(name)
	if zid == id.Invalid {
		return id.Invalid
	}
	entry := fetchdirEntry(entries, zid)
	path := filepath.Join(ds.dirPath, name)

	if ext == "meta" {
		entry.metaSpec = dirMetaSpecFile
		entry.metaPath = path
		return zid
	}
	if entry.contentExt != "" && entry.contentExt != ext {
		entry.duplicates = true
		ds.log.Warn().Str("name", path).Msg("Duplicate content (is ignored)")
		return zid
	}
	if entry.metaSpec != dirMetaSpecFile {
		if ext == "zettel" {
			entry.metaSpec = dirMetaSpecHeader
		} else {
			entry.metaSpec = dirMetaSpecNone
		}
	}
	entry.contentPath = path
	entry.contentExt = ext
	return zid
}

func (ds *dirService) onDeleteFileEvent(entries entrySet, name string) {
	if entries == nil {
		return
	}
	zid, ext := seekZidExt(name)
	if zid == id.Invalid {
		return
	}
	if ext == "meta" {
		if entry, found := entries[zid]; found {
			if entry.metaSpec == dirMetaSpecFile {
				entry.metaSpec = dirMetaSpecNone
				return
			}
		}
	}
	delete(entries, zid)
	ds.notifyChange(box.OnDelete, zid)
}

func (ds *dirService) notifyChange(reason box.UpdateReason, zid id.Zid) {
	if chci := ds.infos; chci != nil {
		ds.log.Trace().Zid(zid).Uint("reason", uint64(reason)).Msg("notifyChange")
		chci <- box.UpdateInfo{Reason: reason, Zid: zid}
	}
}
