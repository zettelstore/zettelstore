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
	Zid      id.Zid                     // zid of the indexed zettel
	backrefs map[id.Zid]bool            // set of back references
	metarefs map[string]map[id.Zid]bool // references to inverse keys
	deadrefs map[id.Zid]bool            // set of dead references
}

// NewZettelIndex creates a new zettel index.
func NewZettelIndex(zid id.Zid) *ZettelIndex {
	return &ZettelIndex{
		Zid:      zid,
		backrefs: make(map[id.Zid]bool),
		metarefs: make(map[string]map[id.Zid]bool),
		deadrefs: make(map[id.Zid]bool),
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
	zi.metarefs[key] = map[id.Zid]bool{zid: true}
}

// AddDeadRef adds a dead reference to a zettel.
func (zi *ZettelIndex) AddDeadRef(zid id.Zid) {
	zi.deadrefs[zid] = true
}

// GetDeadRefs returns all dead references as a sorted list.
func (zi *ZettelIndex) GetDeadRefs() []id.Zid {
	return sortedZids(zi.deadrefs)
}

// GetBackRefs returns all back references as a sorted list.
func (zi *ZettelIndex) GetBackRefs() []id.Zid {
	return sortedZids(zi.backrefs)
}

// GetMetaRefs returns all meta references as a map of strings to a sorted list of references
func (zi *ZettelIndex) GetMetaRefs() map[string][]id.Zid {
	if len(zi.metarefs) == 0 {
		return nil
	}
	result := make(map[string][]id.Zid, len(zi.metarefs))
	for key, refs := range zi.metarefs {
		result[key] = sortedZids(refs)
	}
	return result
}

func sortedZids(refmap map[id.Zid]bool) []id.Zid {
	if l := len(refmap); l > 0 {
		result := make([]id.Zid, 0, l)
		for zid := range refmap {
			result = append(result, zid)
		}
		id.Sort(result)
		return result
	}
	return nil
}
