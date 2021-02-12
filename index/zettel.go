//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package index allows to search for metadata and content.
package index

import (
	"zettelstore.de/z/domain/id"
)

// ZettelIndex contains all index data of a zettel.
type ZettelIndex struct {
	Zid      id.Zid            // zid of the indexed zettel
	backrefs id.Set            // set of back references
	metarefs map[string]id.Set // references to inverse keys
	deadrefs id.Set            // set of dead references
}

// NewZettelIndex creates a new zettel index.
func NewZettelIndex(zid id.Zid) *ZettelIndex {
	return &ZettelIndex{
		Zid:      zid,
		backrefs: id.NewSet(),
		metarefs: make(map[string]id.Set),
		deadrefs: id.NewSet(),
	}
}

// AddBackRef adds a reference to a zettel where the current zettel links to
// without any more information.
func (zi *ZettelIndex) AddBackRef(zid id.Zid) {
	zi.backrefs[zid] = true
}

// AddMetaRef adds a named reference to a zettel. On that zettel, the given
// metadata key should point back to the current zettel.
func (zi *ZettelIndex) AddMetaRef(key string, zid id.Zid) {
	if zids, ok := zi.metarefs[key]; ok {
		zids[zid] = true
		return
	}
	zi.metarefs[key] = id.NewSet(zid)
}

// AddDeadRef adds a dead reference to a zettel.
func (zi *ZettelIndex) AddDeadRef(zid id.Zid) {
	zi.deadrefs[zid] = true
}

// GetDeadRefs returns all dead references as a sorted list.
func (zi *ZettelIndex) GetDeadRefs() []id.Zid {
	return zi.deadrefs.Sort()
}

// GetBackRefs returns all back references as a sorted list.
func (zi *ZettelIndex) GetBackRefs() []id.Zid {
	return zi.backrefs.Sort()
}

// GetMetaRefs returns all meta references as a map of strings to a sorted list of references
func (zi *ZettelIndex) GetMetaRefs() map[string][]id.Zid {
	if len(zi.metarefs) == 0 {
		return nil
	}
	result := make(map[string][]id.Zid, len(zi.metarefs))
	for key, refs := range zi.metarefs {
		result[key] = refs.Sort()
	}
	return result
}
