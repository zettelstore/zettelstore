//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package plaindir manages the directory part of a dirstore.
package plaindir

import (
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/place/dirplace/directory"
)

// plainService specifies a directory service without scanning.
type plainService struct {
	dirPath string
	mx      sync.Mutex
}

// NewService creates a new directory service.
func NewService(directoryPath string) directory.Service {
	return &plainService{
		dirPath: directoryPath,
	}
}

func (ps *plainService) Start() error {
	ps.mx.Lock()
	defer ps.mx.Unlock()
	_, err := os.ReadDir(ps.dirPath)
	return err
}

func (ps *plainService) Stop() error {
	ps.mx.Lock()
	defer ps.mx.Unlock()
	return nil
}

func (ps *plainService) NumEntries() (int, error) {
	ps.mx.Lock()
	defer ps.mx.Unlock()
	entries, err := ps.getEntries()
	if err == nil {
		return len(entries), nil
	}
	return 0, err
}

func (ps *plainService) GetEntries() ([]*directory.Entry, error) {
	ps.mx.Lock()
	defer ps.mx.Unlock()
	entrySet, err := ps.getEntries()
	if err != nil {
		return nil, err
	}
	result := make([]*directory.Entry, 0, len(entrySet))
	for _, entry := range entrySet {
		result = append(result, entry)
	}
	return result, nil
}
func (ps *plainService) getEntries() (map[id.Zid]*directory.Entry, error) {
	dirEntries, err := os.ReadDir(ps.dirPath)
	if err != nil {
		return nil, err
	}
	entrySet := make(map[id.Zid]*directory.Entry)
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}
		if info, err1 := dirEntry.Info(); err1 != nil || !info.Mode().IsRegular() {
			continue
		}
		name := dirEntry.Name()
		match := matchValidFileName(name)
		if len(match) == 0 {
			continue
		}
		zid, err := id.Parse(match[1])
		if err != nil {
			continue
		}
		var entry *directory.Entry
		if e, ok := entrySet[zid]; ok {
			entry = e
		} else {
			entry = &directory.Entry{Zid: zid}
			entrySet[zid] = entry
		}
		updateEntry(entry, filepath.Join(ps.dirPath, name), match[3])
	}
	return entrySet, nil
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

func (ps *plainService) GetEntry(zid id.Zid) (*directory.Entry, error) {
	ps.mx.Lock()
	defer ps.mx.Unlock()
	return ps.getEntry(zid)
}
func (ps *plainService) getEntry(zid id.Zid) (*directory.Entry, error) {
	pattern := filepath.Join(ps.dirPath, zid.String()) + "*.*"
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, nil
	}
	entry := &directory.Entry{Zid: zid}
	for _, path := range paths {
		ext := filepath.Ext(path)
		if len(ext) > 0 && ext[0] == '.' {
			ext = ext[1:]
		}
		updateEntry(entry, path, ext)
	}
	return entry, nil
}

func (ps *plainService) GetNew() (*directory.Entry, error) {
	ps.mx.Lock()
	defer ps.mx.Unlock()
	zid := id.New(false)
	if entry, err := ps.getEntry(zid); entry == nil && err == nil {
		return &directory.Entry{Zid: zid}, nil
	}
	for {
		zid = id.New(true)
		if entry, err := ps.getEntry(zid); entry == nil && err == nil {
			return &directory.Entry{Zid: zid}, nil
		} else if err != nil {
			return nil, err
		}
		// TODO: do not wait here, but in a non-blocking goroutine.
		time.Sleep(100 * time.Millisecond)
	}
}

func (ps *plainService) UpdateEntry(entry *directory.Entry) error {
	ps.mx.Lock()
	defer ps.mx.Unlock()

	// Noting to to, since the actual file update is done by dirplace
	return nil
}

func (ps *plainService) RenameEntry(curEntry, newEntry *directory.Entry) error {
	ps.mx.Lock()
	defer ps.mx.Unlock()

	// Noting to to, since the actual file rename is done by dirplace
	return nil
}

func (ps *plainService) DeleteEntry(zid id.Zid) error {
	ps.mx.Lock()
	defer ps.mx.Unlock()

	// Noting to to, since the actual file delete is done by dirplace
	return nil
}
