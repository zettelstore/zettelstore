//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package notify

import (
	"errors"
	"fmt"
	"regexp"
	"sync"

	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/logger"
	"zettelstore.de/z/search"
)

// DirMetaSpec defines all possibilities where meta data can be stored.
type DirMetaSpec uint8

// Constants for MetaSpec
const (
	_                 DirMetaSpec = iota
	DirMetaSpecNone               // no meta information
	DirMetaSpecFile               // meta information is in meta file
	DirMetaSpecHeader             // meta information is in header
)

// DirEntry stores everything for a directory entry.
type DirEntry struct {
	Zid         id.Zid
	MetaSpec    DirMetaSpec // location of meta information
	Duplicates  bool        // multiple content files
	MetaName    string      // file name of meta information
	ContentName string      // file name of zettel content
	ContentExt  string      // (normalized) file extension of zettel content
}

// IsValid checks whether the entry is valid.
func (e *DirEntry) IsValid() bool {
	return e != nil && e.Zid.IsValid()
}

type entrySet map[id.Zid]*DirEntry

// directoryState signal the internal state of the service.
//
// The following state transitions are possible:
// --newDirService--> dsCreated
// dsCreated --Start--> dsStarting
// dsStarting --last list notification--> dsWorking
// dsWorking --directory missing--> dsMissing
// dsMissing --last list notification--> dsWorking
// --Stop--> dsStopping
type directoryState uint8

const (
	dsCreated  directoryState = iota
	dsStarting                // Reading inital scan
	dsWorking                 // Initial scan complete, fully operational
	dsMissing                 // Directory is missing
	dsStopping                // Service is shut down
)

// DirService specifies a directory service for file based zettel.
type DirService struct {
	log      *logger.Logger
	dirPath  string
	notifier Notifier
	infos    chan<- box.UpdateInfo
	mx       sync.RWMutex // protects status, entries
	state    directoryState
	entries  entrySet
}

// ErrNoDirectory signals missing directory data.
var ErrNoDirectory = errors.New("unable to retrieve zettel directory information")

// NewDirService creates a new directory service.
func NewDirService(log *logger.Logger, notifier Notifier, chci chan<- box.UpdateInfo) *DirService {
	return &DirService{
		log:      log,
		notifier: notifier,
		infos:    chci,
		state:    dsCreated,
	}
}

// Start the directory service.
func (ds *DirService) Start() {
	ds.mx.Lock()
	ds.state = dsStarting
	ds.mx.Unlock()
	go ds.updateEvents()
}

// Refresh the directory entries.
func (ds *DirService) Refresh() {
	ds.notifier.Refresh()
}

// Stop the directory service.
func (ds *DirService) Stop() {
	ds.mx.Lock()
	ds.state = dsStopping
	ds.mx.Unlock()
	ds.notifier.Close()
}

func (ds *DirService) logMissingEntry(action string) error {
	err := ErrNoDirectory
	ds.log.Info().Err(err).Str("action", action).Msg("Unable to get directory information")
	return err
}

// NumDirEntries returns the number of entries in the directory.
func (ds *DirService) NumDirEntries() int {
	ds.mx.RLock()
	defer ds.mx.RUnlock()
	if ds.entries == nil {
		return 0
	}
	return len(ds.entries)
}

// GetDirEntries returns a list of directory entries, which satisfy the given constraint.
func (ds *DirService) GetDirEntries(constraint search.RetrievePredicate) []*DirEntry {
	ds.mx.RLock()
	defer ds.mx.RUnlock()
	if ds.entries == nil {
		return nil
	}
	result := make([]*DirEntry, 0, len(ds.entries))
	for zid, entry := range ds.entries {
		if constraint(zid) {
			copiedEntry := *entry
			result = append(result, &copiedEntry)
		}
	}
	return result
}

// GetDirEntry returns a directory entry with the given zid, or nil if not found.
func (ds *DirService) GetDirEntry(zid id.Zid) *DirEntry {
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

// SetNewDirEntry calculates an empty directory entry with an unused identifier and
// stores it in the directory.
func (ds *DirService) SetNewDirEntry() (id.Zid, error) {
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
	ds.entries[zid] = &DirEntry{Zid: zid}
	return zid, nil
}

// UpdateDirEntry updates an directory entry in place.
func (ds *DirService) UpdateDirEntry(updatedEntry *DirEntry) error {
	entry := *updatedEntry
	ds.mx.Lock()
	defer ds.mx.Unlock()
	if ds.entries == nil {
		return ds.logMissingEntry("update")
	}
	ds.entries[entry.Zid] = &entry
	return nil
}

// RenameDirEntry replaces an existing directory entry with a new one.
func (ds *DirService) RenameDirEntry(oldEntry, newEntry *DirEntry) error {
	ds.mx.Lock()
	defer ds.mx.Unlock()
	if ds.entries == nil {
		return ds.logMissingEntry("rename")
	}
	if _, found := ds.entries[newEntry.Zid]; found {
		return &box.ErrInvalidID{Zid: newEntry.Zid}
	}
	delete(ds.entries, oldEntry.Zid)
	entry := *newEntry
	ds.entries[entry.Zid] = &entry
	return nil
}

// DeleteDirEntry removes a entry from the directory.
func (ds *DirService) DeleteDirEntry(zid id.Zid) error {
	ds.mx.Lock()
	defer ds.mx.Unlock()
	if ds.entries == nil {
		return ds.logMissingEntry("delete")
	}
	delete(ds.entries, zid)
	return nil
}

func (ds *DirService) updateEvents() {
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
		case Error:
			newEntries = nil
			if state != dsMissing {
				ds.log.Warn().Err(ev.Err).Msg("Notifier confused")
			}
		case Make:
			newEntries = make(entrySet)
		case List:
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
		case Destroy:
			newEntries = nil
			ds.onDestroyDirectory()
			ds.log.Error().Str("path", ds.dirPath).Msg("Zettel directory missing")
		case Update:
			ds.mx.Lock()
			zid := ds.onUpdateFileEvent(ds.entries, ev.Name)
			ds.mx.Unlock()
			if zid != id.Invalid {
				ds.notifyChange(box.OnUpdate, zid)
			}
		case Delete:
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

func (ds *DirService) onCreateDirectory(zids id.Slice, prevEntries entrySet) {
	for _, zid := range zids {
		ds.notifyChange(box.OnUpdate, zid)
		delete(prevEntries, zid)
	}

	// These were previously stored, by are not found now.
	// Notify system that these were deleted, e.g. for updating the index.
	for zid := range prevEntries {
		ds.notifyChange(box.OnDelete, zid)
	}
}

func (ds *DirService) onDestroyDirectory() {
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

func fetchdirEntry(entries entrySet, zid id.Zid) *DirEntry {
	if entry, found := entries[zid]; found {
		return entry
	}
	entry := &DirEntry{Zid: zid}
	entries[zid] = entry
	return entry
}

func (ds *DirService) onUpdateFileEvent(entries entrySet, name string) id.Zid {
	if entries == nil {
		return id.Invalid
	}
	zid, ext := seekZidExt(name)
	if zid == id.Invalid {
		return id.Invalid
	}
	entry := fetchdirEntry(entries, zid)

	if ext == "meta" {
		entry.MetaSpec = DirMetaSpecFile
		entry.MetaName = name
		return zid
	}
	if entry.ContentExt != "" && entry.ContentExt != ext {
		entry.Duplicates = true
		ds.log.Warn().Str("name", name).Msg("Duplicate content (is ignored)")
		return zid
	}
	if entry.MetaSpec != DirMetaSpecFile {
		if ext == "zettel" {
			entry.MetaSpec = DirMetaSpecHeader
		} else {
			entry.MetaSpec = DirMetaSpecNone
		}
	}
	entry.ContentName = name
	entry.ContentExt = ext
	return zid
}

func (ds *DirService) onDeleteFileEvent(entries entrySet, name string) {
	if entries == nil {
		return
	}
	zid, ext := seekZidExt(name)
	if zid == id.Invalid {
		return
	}
	if ext == "meta" {
		if entry, found := entries[zid]; found {
			if entry.MetaSpec == DirMetaSpecFile {
				entry.MetaSpec = DirMetaSpecNone
				return
			}
		}
	}
	delete(entries, zid)
	ds.notifyChange(box.OnDelete, zid)
}

func (ds *DirService) notifyChange(reason box.UpdateReason, zid id.Zid) {
	if chci := ds.infos; chci != nil {
		ds.log.Trace().Zid(zid).Uint("reason", uint64(reason)).Msg("notifyChange")
		chci <- box.UpdateInfo{Reason: reason, Zid: zid}
	}
}
