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
	"sync"

	"zettelstore.de/z/domain/id"
)

type arAction int

const (
	arNothing arAction = iota
	arReload
	arUpdate
	arDelete
)

type anteroom struct {
	next    *anteroom
	waiting map[id.Zid]arAction
	curLoad int
	reload  bool
}

type anterooms struct {
	mx      sync.Mutex
	first   *anteroom
	last    *anteroom
	maxLoad int
}

func newAnterooms(maxLoad int) *anterooms {
	return &anterooms{maxLoad: maxLoad}
}

func (ar *anterooms) Enqueue(zid id.Zid, action arAction) {
	if !zid.IsValid() {
		return
	}
	ar.mx.Lock()
	defer ar.mx.Unlock()
	if ar.first == nil {
		ar.first = ar.makeAnteroom(zid, action)
		ar.last = ar.first
		return
	}
	for room := ar.first; room != nil; room = room.next {
		if room.reload {
			continue // Do not place zettel in reload room
		}
		if a, ok := room.waiting[zid]; ok {
			if action == a {
				return
			}
			room.waiting[zid] = action
			return
		}
	}
	if room := ar.last; !room.reload && (ar.maxLoad == 0 || room.curLoad < ar.maxLoad) {
		room.waiting[zid] = action
		room.curLoad++
		return
	}
	room := ar.makeAnteroom(zid, action)
	ar.last.next = room
	ar.last = room
}

func (ar *anterooms) makeAnteroom(zid id.Zid, action arAction) *anteroom {
	cap := ar.maxLoad
	if cap == 0 {
		cap = 100
	}
	waiting := make(map[id.Zid]arAction, cap)
	waiting[zid] = action
	return &anteroom{next: nil, waiting: waiting, curLoad: 1, reload: false}
}

func (ar *anterooms) Reset() {
	ar.mx.Lock()
	defer ar.mx.Unlock()
	ar.first = ar.makeAnteroom(id.Invalid, arReload)
	ar.last = ar.first
}

func (ar *anterooms) Reload(delZids []id.Zid, newZids id.Set) {
	ar.mx.Lock()
	defer ar.mx.Unlock()
	delWaiting := make(map[id.Zid]arAction, len(delZids))
	for _, zid := range delZids {
		if zid.IsValid() {
			delWaiting[zid] = arDelete
		}
	}
	newWaiting := make(map[id.Zid]arAction, len(newZids))
	for zid := range newZids {
		if zid.IsValid() {
			newWaiting[zid] = arUpdate
		}
	}

	// Delete previous reload rooms
	room := ar.first
	for ; room != nil && room.reload; room = room.next {
	}
	ar.first = room
	if room == nil {
		ar.last = nil
	}

	if ds := len(delWaiting); ds > 0 {
		if ns := len(newWaiting); ns > 0 {
			roomNew := &anteroom{next: ar.first, waiting: newWaiting, curLoad: ns, reload: true}
			ar.first = &anteroom{next: roomNew, waiting: delWaiting, curLoad: ds, reload: true}
			if roomNew.next == nil {
				ar.last = roomNew
			}
		} else {
			ar.first = &anteroom{next: ar.first, waiting: delWaiting, curLoad: ds}
			if ar.first.next == nil {
				ar.last = ar.first
			}
		}
	} else {
		if ns := len(newWaiting); ns > 0 {
			ar.first = &anteroom{next: ar.first, waiting: newWaiting, curLoad: ns}
			if ar.first.next == nil {
				ar.last = ar.first
			}
		} else {
			ar.first = nil
			ar.last = nil
		}
	}
}

func (ar *anterooms) Dequeue() (arAction, id.Zid) {
	ar.mx.Lock()
	defer ar.mx.Unlock()
	if ar.first == nil {
		return arNothing, id.Invalid
	}
	for zid, action := range ar.first.waiting {
		delete(ar.first.waiting, zid)
		if len(ar.first.waiting) == 0 {
			ar.first = ar.first.next
			if ar.first == nil {
				ar.last = nil
			}
		}
		return action, zid
	}
	return arNothing, id.Invalid
}
