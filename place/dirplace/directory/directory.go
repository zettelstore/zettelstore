//-----------------------------------------------------------------------------
// Copyright (c) 2020-2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package directory manages the directory interface of a dirstore.
package directory

import "zettelstore.de/z/domain/id"

// Service is the interface of a directory service.
type Service interface {
	Start() error
	Stop() error
	NumEntries() (int, error)
	GetEntries() ([]Entry, error)
	GetEntry(zid id.Zid) (Entry, error)
	GetNew() (Entry, error)
	UpdateEntry(entry *Entry) error
	RenameEntry(curEntry, newEntry *Entry) error
	DeleteEntry(zid id.Zid) error
}

// MetaSpec defines all possibilities where meta data can be stored.
type MetaSpec int

// Constants for MetaSpec
const (
	_              MetaSpec = iota
	MetaSpecNone            // no meta information
	MetaSpecFile            // meta information is in meta file
	MetaSpecHeader          // meta information is in header
)

// Entry stores everything for a directory entry.
type Entry struct {
	Zid         id.Zid
	MetaSpec    MetaSpec // location of meta information
	MetaPath    string   // file path of meta information
	ContentPath string   // file path of zettel content
	ContentExt  string   // (normalized) file extension of zettel content
	Duplicates  bool     // multiple content files
}

// IsValid checks whether the entry is valid.
func (e *Entry) IsValid() bool {
	return e.Zid.IsValid()
}
