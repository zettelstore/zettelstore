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
	"fmt"
	"time"

	"zettelstore.de/z/domain/id"
	"zettelstore.de/z/place/change"
	"zettelstore.de/z/place/dirplace/directory"
)

// plainService specifies a directory service without scanning.
type plainService struct {
	dirPath    string
	rescanTime time.Duration
	infos      chan<- change.Info
}

// NewService creates a new directory service.
func NewService(directoryPath string, rescanTime time.Duration, chci chan<- change.Info) directory.Service {
	return &plainService{
		dirPath:    directoryPath,
		rescanTime: rescanTime,
		infos:      chci,
	}
}

func (ps *plainService) Start() error {
	return nil
}

func (ps *plainService) Stop() error {
	return nil
}

func (ps *plainService) NumEntries() (int, error) {
	return 0, nil
}

func (ps *plainService) GetEntries() ([]*directory.Entry, error) {
	return nil, nil
}

func (ps *plainService) GetEntry(zid id.Zid) (*directory.Entry, error) {
	return nil, fmt.Errorf("NYI")
}

func (ps *plainService) GetNew() (*directory.Entry, error) {
	return nil, fmt.Errorf("NYI")
}

func (ps *plainService) UpdateEntry(entry *directory.Entry) error {
	return nil
}

func (ps *plainService) RenameEntry(curEntry, newEntry *directory.Entry) error {
	return nil
}

func (ps *plainService) DeleteEntry(zid id.Zid) error {
	// Noting to to, since the actual file delete is done by dirplace
	return nil
}
