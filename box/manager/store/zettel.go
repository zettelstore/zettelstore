//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//
// SPDX-License-Identifier: EUPL-1.2
// SPDX-FileCopyrightText: 2021-present Detlef Stern
//-----------------------------------------------------------------------------

package store

import (
	"zettelstore.de/z/zettel/id"
	"zettelstore.de/z/zettel/meta"
)

// ZettelIndex contains all index data of a zettel.
type ZettelIndex struct {
	Zid         id.ZidO            // zid of the indexed zettel
	meta        *meta.Meta         // full metadata
	backrefs    id.SetO            // set of back references
	inverseRefs map[string]id.SetO // references of inverse keys
	deadrefs    id.SetO            // set of dead references
	words       WordSet
	urls        WordSet
}

// NewZettelIndex creates a new zettel index.
func NewZettelIndex(m *meta.Meta) *ZettelIndex {
	return &ZettelIndex{
		Zid:         m.ZidO,
		meta:        m,
		backrefs:    id.NewSetO(),
		inverseRefs: make(map[string]id.SetO),
		deadrefs:    id.NewSetO(),
	}
}

// AddBackRef adds a reference to a zettel where the current zettel links to
// without any more information.
func (zi *ZettelIndex) AddBackRef(zid id.ZidO) {
	zi.backrefs.Add(zid)
}

// AddInverseRef adds a named reference to a zettel. On that zettel, the given
// metadata key should point back to the current zettel.
func (zi *ZettelIndex) AddInverseRef(key string, zid id.ZidO) {
	if zids, ok := zi.inverseRefs[key]; ok {
		zids.Add(zid)
		return
	}
	zi.inverseRefs[key] = id.NewSetO(zid)
}

// AddDeadRef adds a dead reference to a zettel.
func (zi *ZettelIndex) AddDeadRef(zid id.ZidO) {
	zi.deadrefs.Add(zid)
}

// SetWords sets the words to the given value.
func (zi *ZettelIndex) SetWords(words WordSet) { zi.words = words }

// SetUrls sets the words to the given value.
func (zi *ZettelIndex) SetUrls(urls WordSet) { zi.urls = urls }

// GetDeadRefs returns all dead references as a sorted list.
func (zi *ZettelIndex) GetDeadRefs() id.SliceO { return zi.deadrefs.Sorted() }

// GetMeta return just the raw metadata.
func (zi *ZettelIndex) GetMeta() *meta.Meta { return zi.meta }

// GetBackRefs returns all back references as a sorted list.
func (zi *ZettelIndex) GetBackRefs() id.SliceO { return zi.backrefs.Sorted() }

// GetInverseRefs returns all inverse meta references as a map of strings to a sorted list of references
func (zi *ZettelIndex) GetInverseRefs() map[string]id.SliceO {
	if len(zi.inverseRefs) == 0 {
		return nil
	}
	result := make(map[string]id.SliceO, len(zi.inverseRefs))
	for key, refs := range zi.inverseRefs {
		result[key] = refs.Sorted()
	}
	return result
}

// GetWords returns a reference to the set of words. It must not be modified.
func (zi *ZettelIndex) GetWords() WordSet { return zi.words }

// GetUrls returns a reference to the set of URLs. It must not be modified.
func (zi *ZettelIndex) GetUrls() WordSet { return zi.urls }
