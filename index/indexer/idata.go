//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package indexer allows to search for metadata and content.
package indexer

import (
	"zettelstore.de/z/domain/id"
)

type indexData struct {
	Zid       id.Zid                     // zid of the indexed zettel
	deadlinks map[id.Zid]bool            // set of dead links
	backlinks map[id.Zid]bool            // set of backlinks
	links     map[string]map[id.Zid]bool // links to inverse keys
}

func newIndexData(zid id.Zid) *indexData {
	return &indexData{
		Zid:       zid,
		deadlinks: make(map[id.Zid]bool),
		backlinks: make(map[id.Zid]bool),
		links:     make(map[string]map[id.Zid]bool),
	}
}

func (i *indexData) AddDeadlink(zid id.Zid) {
	// log.Println("INDX", i.Zid, "DEAD", zid)
	i.deadlinks[zid] = true
}
func (i *indexData) AddBacklink(zid id.Zid) {
	// log.Println("INDX", i.Zid, "BACK", zid)
	i.backlinks[zid] = true
}
func (i *indexData) AddLink(key string, zid id.Zid) {
	// log.Println("INDX", i.Zid, "META", zid, key)
	if zids, ok := i.links[key]; ok {
		zids[zid] = true
		return
	}
	i.links[key] = map[id.Zid]bool{zid: true}
}
