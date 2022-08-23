//-----------------------------------------------------------------------------
// Copyright (c) 2020-2022 Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package notify

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"zettelstore.de/z/box"
	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/logger"
	"zettelstore.de/z/parser"
	"zettelstore.de/z/search"
	"zettelstore.de/z/strfun"
)

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
func (ds *DirService) RenameDirEntry(oldEntry *DirEntry, newZid id.Zid) (DirEntry, error) {
	ds.mx.Lock()
	defer ds.mx.Unlock()
	if ds.entries == nil {
		return DirEntry{}, ds.logMissingEntry("rename")
	}
	if _, found := ds.entries[newZid]; found {
		return DirEntry{}, &box.ErrInvalidID{Zid: newZid}
	}
	oldZid := oldEntry.Zid
	newEntry := DirEntry{
		Zid:         newZid,
		MetaName:    renameFilename(oldEntry.MetaName, oldZid, newZid),
		ContentName: renameFilename(oldEntry.ContentName, oldZid, newZid),
		ContentExt:  oldEntry.ContentExt,
		// Duplicates must not be set, because duplicates will be deleted
	}
	delete(ds.entries, oldZid)
	ds.entries[newZid] = &newEntry
	return newEntry, nil
}

func renameFilename(name string, curID, newID id.Zid) string {
	if cur := curID.String(); strings.HasPrefix(name, cur) {
		name = newID.String() + name[len(cur):]
	}
	return name
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
				ds.notifyChange(box.OnZettel, zid)
			}
		case Delete:
			ds.mx.Lock()
			zid := ds.onDeleteFileEvent(ds.entries, ev.Name)
			ds.mx.Unlock()
			if zid != id.Invalid {
				ds.notifyChange(box.OnZettel, zid)
			}
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
		ds.notifyChange(box.OnZettel, zid)
		delete(prevEntries, zid)
	}

	// These were previously stored, by are not found now.
	// Notify system that these were deleted, e.g. for updating the index.
	for zid := range prevEntries {
		ds.notifyChange(box.OnZettel, zid)
	}
}

func (ds *DirService) onDestroyDirectory() {
	ds.mx.Lock()
	entries := ds.entries
	ds.entries = nil
	ds.state = dsMissing
	ds.mx.Unlock()
	for zid := range entries {
		ds.notifyChange(box.OnZettel, zid)
	}
}

var validFileName = regexp.MustCompile(`^(\d{14})`)

func matchValidFileName(name string) []string {
	return validFileName.FindStringSubmatch(name)
}

func seekZid(name string) id.Zid {
	match := matchValidFileName(name)
	if len(match) == 0 {
		return id.Invalid
	}
	zid, err := id.Parse(match[1])
	if err != nil {
		return id.Invalid
	}
	return zid
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
	zid := seekZid(name)
	if zid == id.Invalid {
		return id.Invalid
	}
	entry := fetchdirEntry(entries, zid)
	dupName1, dupName2 := ds.updateEntry(entry, name)
	if dupName1 != "" {
		ds.log.Warn().Str("name", dupName1).Msg("Duplicate content (is ignored)")
		if dupName2 != "" {
			ds.log.Warn().Str("name", dupName2).Msg("Duplicate content (is ignored)")
		}
	}
	return zid
}

func (ds *DirService) onDeleteFileEvent(entries entrySet, name string) id.Zid {
	if entries == nil {
		return id.Invalid
	}
	zid := seekZid(name)
	if zid == id.Invalid {
		return id.Invalid
	}
	entry, found := entries[zid]
	if !found {
		return zid
	}
	for i, dupName := range entry.UselessFiles {
		if dupName == name {
			removeDuplicate(entry, i)
			return zid
		}
	}
	if name == entry.ContentName {
		entry.ContentName = ""
		entry.ContentExt = ""
		ds.replayUpdateUselessFiles(entry)
	} else if name == entry.MetaName {
		entry.MetaName = ""
		ds.replayUpdateUselessFiles(entry)
	}
	if entry.ContentName == "" && entry.MetaName == "" {
		delete(entries, zid)
	}
	return zid
}

func removeDuplicate(entry *DirEntry, i int) {
	if len(entry.UselessFiles) == 1 {
		entry.UselessFiles = nil
		return
	}
	entry.UselessFiles = entry.UselessFiles[:i+copy(entry.UselessFiles[i:], entry.UselessFiles[i+1:])]
}

func (ds *DirService) replayUpdateUselessFiles(entry *DirEntry) {
	uselessFiles := entry.UselessFiles
	if len(uselessFiles) == 0 {
		return
	}
	entry.UselessFiles = make([]string, 0, len(uselessFiles))
	for _, name := range uselessFiles {
		ds.updateEntry(entry, name)
	}
	if len(uselessFiles) == len(entry.UselessFiles) {
		return
	}
loop:
	for _, prevName := range uselessFiles {
		for _, newName := range entry.UselessFiles {
			if prevName == newName {
				continue loop
			}
		}
		ds.log.Info().Str("name", prevName).Msg("Previous duplicate file becomes useful")
	}
}

func (ds *DirService) updateEntry(entry *DirEntry, name string) (string, string) {
	ext := onlyExt(name)
	if !extIsMetaAndContent(entry.ContentExt) {
		if ext == "" {
			return updateEntryMeta(entry, name), ""
		}
		if entry.MetaName == "" {
			if nameWithoutExt(name, ext) == entry.ContentName {
				// We have marked a file as content file, but it is a metadata file,
				// because it is the same as the new file without extension.
				entry.MetaName = entry.ContentName
				entry.ContentName = ""
				entry.ContentExt = ""
				ds.replayUpdateUselessFiles(entry)
			} else if entry.ContentName != "" && nameWithoutExt(entry.ContentName, entry.ContentExt) == name {
				// We have already a valid content file, and new file should serve as metadata file,
				// because it is the same as the content file without extension.
				entry.MetaName = name
				return "", ""
			}
		}
	}
	return updateEntryContent(entry, name, ext)
}

func nameWithoutExt(name, ext string) string {
	return name[0 : len(name)-len(ext)-1]
}

func updateEntryMeta(entry *DirEntry, name string) string {
	metaName := entry.MetaName
	if metaName == "" {
		entry.MetaName = name
		return ""
	}
	if metaName == name {
		return ""
	}
	if newNameIsBetter(metaName, name) {
		entry.MetaName = name
		return addUselessFile(entry, metaName)
	}
	return addUselessFile(entry, name)
}

func updateEntryContent(entry *DirEntry, name, ext string) (string, string) {
	contentName := entry.ContentName
	if contentName == "" {
		entry.ContentName = name
		entry.ContentExt = ext
		return "", ""
	}
	if contentName == name {
		return "", ""
	}
	contentExt := entry.ContentExt
	if contentExt == ext {
		if newNameIsBetter(contentName, name) {
			entry.ContentName = name
			return addUselessFile(entry, contentName), ""
		}
		return addUselessFile(entry, name), ""
	}
	if contentExt == extZettel {
		return addUselessFile(entry, name), ""
	}
	if ext == extZettel {
		entry.ContentName = name
		entry.ContentExt = ext
		contentName = addUselessFile(entry, contentName)
		if metaName := entry.MetaName; metaName != "" {
			metaName = addUselessFile(entry, metaName)
			entry.MetaName = ""
			return contentName, metaName
		}
		return contentName, ""
	}
	if newExtIsBetter(contentExt, ext) {
		entry.ContentName = name
		entry.ContentExt = ext
		return addUselessFile(entry, contentName), ""
	}
	return addUselessFile(entry, name), ""
}
func addUselessFile(entry *DirEntry, name string) string {
	for _, dupName := range entry.UselessFiles {
		if name == dupName {
			return ""
		}
	}
	entry.UselessFiles = append(entry.UselessFiles, name)
	return name
}

func onlyExt(name string) string {
	ext := filepath.Ext(name)
	if ext == "" || ext[0] != '.' {
		return ext
	}
	return ext[1:]
}

func newNameIsBetter(oldName, newName string) bool {
	if len(oldName) < len(newName) {
		return false
	}
	return oldName > newName
}

var supportedSyntax, primarySyntax strfun.Set

func init() {
	syntaxList := parser.GetSyntaxes()
	supportedSyntax = strfun.NewSet(syntaxList...)
	primarySyntax = make(map[string]struct{}, len(syntaxList))
	for _, syntax := range syntaxList {
		if parser.Get(syntax).Name == syntax {
			primarySyntax.Set(syntax)
		}
	}
}
func newExtIsBetter(oldExt, newExt string) bool {
	oldSyntax := supportedSyntax.Has(oldExt)
	if oldSyntax != supportedSyntax.Has(newExt) {
		return !oldSyntax
	}
	if oldSyntax {
		if oldExt == "zmk" {
			return false
		}
		if newExt == "zmk" {
			return true
		}
		oldInfo := parser.Get(oldExt)
		newInfo := parser.Get(newExt)
		if oldTextParser := oldInfo.IsTextParser; oldTextParser != newInfo.IsTextParser {
			return !oldTextParser
		}
		if oldImageFormat := oldInfo.IsImageFormat; oldImageFormat != newInfo.IsImageFormat {
			return oldImageFormat
		}
		if oldPrimary := primarySyntax.Has(oldExt); oldPrimary != primarySyntax.Has(newExt) {
			return !oldPrimary
		}
	}

	oldLen := len(oldExt)
	newLen := len(newExt)
	if oldLen != newLen {
		return newLen < oldLen
	}
	return newExt < oldExt
}

func (ds *DirService) notifyChange(reason box.UpdateReason, zid id.Zid) {
	if chci := ds.infos; chci != nil {
		ds.log.Trace().Zid(zid).Uint("reason", uint64(reason)).Msg("notifyChange")
		chci <- box.UpdateInfo{Reason: reason, Zid: zid}
	}
}
