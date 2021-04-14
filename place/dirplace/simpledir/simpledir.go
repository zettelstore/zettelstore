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
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/place"
	"zettelstore.de/z/place/dirplace/directory"
)

// simpleService specifies a directory service without scanning.
type simpleService struct {
	dirPath string
	mx      sync.Mutex
}

// NewService creates a new directory service.
func NewService(directoryPath string) directory.Service {
	return &simpleService{
		dirPath: directoryPath,
	}
}

func (ss *simpleService) Start() error {
	ss.mx.Lock()
	defer ss.mx.Unlock()
	_, err := os.ReadDir(ss.dirPath)
	return err
}

func (ss *simpleService) Stop() error {
	return nil
}

func (ss *simpleService) NumEntries() (int, error) {
	ss.mx.Lock()
	defer ss.mx.Unlock()
	entries, err := ss.getEntries()
	if err == nil {
		return len(entries), nil
	}
	return 0, err
}

func (ss *simpleService) GetEntries() ([]*directory.Entry, error) {
	ss.mx.Lock()
	defer ss.mx.Unlock()
	entrySet, err := ss.getEntries()
	if err != nil {
		return nil, err
	}
	result := make([]*directory.Entry, 0, len(entrySet))
	for _, entry := range entrySet {
		result = append(result, entry)
	}
	return result, nil
}
func (ss *simpleService) getEntries() (map[id.Zid]*directory.Entry, error) {
	dirEntries, err := os.ReadDir(ss.dirPath)
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
		updateEntry(entry, filepath.Join(ss.dirPath, name), match[3])
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

func (ss *simpleService) GetEntry(zid id.Zid) (*directory.Entry, error) {
	ss.mx.Lock()
	defer ss.mx.Unlock()
	return ss.getEntry(zid)
}
func (ss *simpleService) getEntry(zid id.Zid) (*directory.Entry, error) {
	pattern := filepath.Join(ss.dirPath, zid.String()) + "*.*"
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

func (ss *simpleService) GetNew() (*directory.Entry, error) {
	ss.mx.Lock()
	defer ss.mx.Unlock()
	zid, err := place.GetNewZid(func(zid id.Zid) (bool, error) {
		entry, err := ss.getEntry(zid)
		if err != nil {
			return false, nil
		}
		return !entry.IsValid(), nil
	})
	if err != nil {
		return nil, err
	}
	return &directory.Entry{Zid: zid}, nil
}

func (ss *simpleService) UpdateEntry(entry *directory.Entry) error {
	// Noting to to, since the actual file update is done by dirplace
	return nil
}

func (ss *simpleService) RenameEntry(curEntry, newEntry *directory.Entry) error {
	// Noting to to, since the actual file rename is done by dirplace
	return nil
}

func (ss *simpleService) DeleteEntry(zid id.Zid) error {
	// Noting to to, since the actual file delete is done by dirplace
	return nil
}
