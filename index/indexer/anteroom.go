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
	if !zid.IsValid() || action == arNothing || action == arReload {
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
		a, ok := room.waiting[zid]
		if !ok {
			continue
		}
		switch action {
		case a:
			return
		case arUpdate:
			room.waiting[zid] = action
		case arDelete:
			room.waiting[zid] = action
		}
		return
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
	c := ar.maxLoad
	if c == 0 {
		c = 100
	}
	waiting := make(map[id.Zid]arAction, c)
	waiting[zid] = action
	return &anteroom{next: nil, waiting: waiting, curLoad: 1, reload: false}
}

func (ar *anterooms) Reset() {
	ar.mx.Lock()
	defer ar.mx.Unlock()
	ar.first = ar.makeAnteroom(id.Invalid, arReload)
	ar.last = ar.first
}

func (ar *anterooms) Reload(delZids id.Slice, newZids id.Set) {
	ar.mx.Lock()
	defer ar.mx.Unlock()
	delWaiting := createWaitingSlice(delZids, arDelete)
	newWaiting := createWaitingSet(newZids, arUpdate)
	ar.deleteReloadedRooms()

	if ds := len(delWaiting); ds > 0 {
		if ns := len(newWaiting); ns > 0 {
			roomNew := &anteroom{next: ar.first, waiting: newWaiting, curLoad: ns, reload: true}
			ar.first = &anteroom{next: roomNew, waiting: delWaiting, curLoad: ds, reload: true}
			if roomNew.next == nil {
				ar.last = roomNew
			}
			return
		}

		ar.first = &anteroom{next: ar.first, waiting: delWaiting, curLoad: ds}
		if ar.first.next == nil {
			ar.last = ar.first
		}
		return
	}

	if ns := len(newWaiting); ns > 0 {
		ar.first = &anteroom{next: ar.first, waiting: newWaiting, curLoad: ns}
		if ar.first.next == nil {
			ar.last = ar.first
		}
		return
	}

	ar.first = nil
	ar.last = nil
}

func createWaitingSlice(zids id.Slice, action arAction) map[id.Zid]arAction {
	waitingSet := make(map[id.Zid]arAction, len(zids))
	for _, zid := range zids {
		if zid.IsValid() {
			waitingSet[zid] = action
		}
	}
	return waitingSet
}

func createWaitingSet(zids id.Set, action arAction) map[id.Zid]arAction {
	waitingSet := make(map[id.Zid]arAction, len(zids))
	for zid := range zids {
		if zid.IsValid() {
			waitingSet[zid] = action
		}
	}
	return waitingSet
}

func (ar *anterooms) deleteReloadedRooms() {
	room := ar.first
	for room != nil && room.reload {
		room = room.next
	}
	ar.first = room
	if room == nil {
		ar.last = nil
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
