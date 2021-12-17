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
	"errors"
	"fmt"
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
	metaName    string      // file name of meta information
	contentName string      // file name of zettel content
	contentExt  string      // (normalized) file extension of zettel content
}

// isValid checks whether the entry is valid.
func (e *dirEntry) isValid() bool {
	return e != nil && e.zid.IsValid()
}

type entrySet map[id.Zid]*dirEntry

// directoryState signal the internal state of the service.
//
// The following state transitions are possible:
// --newDirService--> dsCreated
// dsCreated --startDirService--> dsStarting
// dsStarting --last list notification--> dsWorking
// dsWorking --directory missing--> dsMissing
// dsMissing --last list notification--> dsWorking
// --stopDirService--> dsStopping
type directoryState uint8

const (
	dsCreated  directoryState = iota
	dsStarting                // Reading inital scan
	dsWorking                 // Initial scan complete, fully operational
	dsMissing                 // Directory is missing
	dsStopping                // Service is shut down
)

// dirService specifies a directory service for file based zettel.
type dirService struct {
	log      *logger.Logger
	dirPath  string
	notifier notify.Notifier
	infos    chan<- box.UpdateInfo
	mx       sync.RWMutex // protects status, entries
	state    directoryState
	entries  entrySet
}

var ErrNoDirectory = errors.New("no zettel directory found")

// newDirService creates a new directory service.
func newDirService(log *logger.Logger, directoryPath string, notifier notify.Notifier, chci chan<- box.UpdateInfo) *dirService {
	return &dirService{
		log:      log,
		dirPath:  directoryPath,
		notifier: notifier,
		infos:    chci,
		state:    dsCreated,
	}
}

func (ds *dirService) startDirService() {
	ds.mx.Lock()
	ds.state = dsStarting
	ds.mx.Unlock()
	go ds.updateEvents()
}

func (ds *dirService) refreshDirService() {
	ds.notifier.Refresh()
}

func (ds *dirService) stopDirService() {
	ds.mx.Lock()
	ds.state = dsStopping
	ds.mx.Unlock()
	ds.notifier.Close()
}

func (ds *dirService) logMissingEntry(action string) error {
	err := ErrNoDirectory
	ds.log.Info().Err(err).Str("action", action).Msg("Unable to get directory information")
	return err
}

func (ds *dirService) countDirEntries() int {
	ds.mx.RLock()
	defer ds.mx.RUnlock()
	if ds.entries == nil {
		return 0
	}
	return len(ds.entries)
}

func (ds *dirService) getDirEntries() []*dirEntry {
	ds.mx.RLock()
	defer ds.mx.RUnlock()
	if ds.entries == nil {
		return nil
	}
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
	if ds.entries == nil {
		return nil
	}
	foundEntry := ds.entries[zid]
	if foundEntry == nil {
		return nil
	}
	result := *foundEntry
	return &result
}

func (ds *dirService) calcNewDirEntry() (id.Zid, error) {
	ds.mx.Lock()
	defer ds.mx.Unlock()
	if ds.entries == nil {
		return id.Invalid, ds.logMissingEntry("new")
	}
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

func (ds *dirService) updateDirEntry(updatedEntry *dirEntry) error {
	entry := *updatedEntry
	ds.mx.Lock()
	defer ds.mx.Unlock()
	if ds.entries == nil {
		return ds.logMissingEntry("update")
	}
	ds.entries[entry.zid] = &entry
	return nil
}

func (ds *dirService) renameDirEntry(oldEntry, newEntry *dirEntry) error {
	ds.mx.Lock()
	defer ds.mx.Unlock()
	if ds.entries == nil {
		return ds.logMissingEntry("rename")
	}
	if _, found := ds.entries[newEntry.zid]; found {
		return &box.ErrInvalidID{Zid: newEntry.zid}
	}
	delete(ds.entries, oldEntry.zid)
	entry := *newEntry
	ds.entries[entry.zid] = &entry
	return nil
}

func (ds *dirService) deleteDirEntry(zid id.Zid) error {
	ds.mx.Lock()
	defer ds.mx.Unlock()
	if ds.entries == nil {
		return ds.logMissingEntry("delete")
	}
	delete(ds.entries, zid)
	return nil
}

func (ds *dirService) updateEvents() {
	var newEntries entrySet
	for ev := range ds.notifier.Events() {
		ds.mx.RLock()
		state := ds.state
		ds.mx.RUnlock()

		if msg := ds.log.Trace(); msg.Enabled() {
			msg.Uint("state", uint64(state)).Str("op", ev.Op.String()).Str("name", ev.Name).Msg("notifyEvent")
		}
		if state == dsStopping {
			break
		}

		switch ev.Op {
		case notify.Error:
			newEntries = nil
			if state != dsMissing {
				ds.log.Warn().Err(ev.Err).Msg("Notifier confused")
			}
		case notify.Make:
			newEntries = make(entrySet)
		case notify.List:
			if ev.Name == "" {
				zids := getNewZids(newEntries)
				ds.mx.Lock()
				fromMissing := ds.state == dsMissing
				prevEntries := ds.entries
				ds.entries = newEntries
				ds.state = dsWorking
				ds.mx.Unlock()
				newEntries = nil
				ds.onCreateDirectory(zids, prevEntries)
				if fromMissing {
					ds.log.Info().Str("path", ds.dirPath).Msg("Zettel directory found")
				}
			} else if newEntries != nil {
				ds.onUpdateFileEvent(newEntries, ev.Name)
			}
		case notify.Destroy:
			newEntries = nil
			ds.onDestroyDirectory()
			ds.log.Error().Str("path", ds.dirPath).Msg("Zettel directory missing")
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
			ds.log.Warn().Str("event", fmt.Sprintf("%v", ev)).Msg("Unknown zettel notification event")
		}
	}
}

func getNewZids(entries entrySet) id.Slice {
	zids := make(id.Slice, 0, len(entries))
	for zid := range entries {
		zids = append(zids, zid)
	}
	return zids
}

func (ds *dirService) onCreateDirectory(zids id.Slice, prevEntries entrySet) {
	for _, zid := range zids {
		ds.notifyChange(box.OnUpdate, zid)
		delete(prevEntries, zid)
	}

	// These were previously stored, by are not found now.
	// Notify system that these were deleted, e.g. for updating the index.
	for zid := range prevEntries {
		ds.notifyChange(box.OnDelete, zid)
	}

	// This may be not needed any more.
	ds.notifyChange(box.OnReload, id.Invalid)
}

func (ds *dirService) onDestroyDirectory() {
	ds.mx.Lock()
	entries := ds.entries
	ds.entries = nil
	ds.state = dsMissing
	ds.mx.Unlock()
	for zid := range entries {
		ds.notifyChange(box.OnDelete, zid)
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

	if ext == "meta" {
		entry.metaSpec = dirMetaSpecFile
		entry.metaName = name
		return zid
	}
	if entry.contentExt != "" && entry.contentExt != ext {
		entry.duplicates = true
		ds.log.Warn().Str("name", name).Msg("Duplicate content (is ignored)")
		return zid
	}
	if entry.metaSpec != dirMetaSpecFile {
		if ext == "zettel" {
			entry.metaSpec = dirMetaSpecHeader
		} else {
			entry.metaSpec = dirMetaSpecNone
		}
	}
	entry.contentName = name
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
