//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package manager coordinates the various boxes and indexes of a Zettelstore.
package manager

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
	num     uint64
	next    *anteroom
	waiting map[id.Zid]arAction
	curLoad int
	reload  bool
}

type anterooms struct {
	mx      sync.Mutex
	nextNum uint64
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
			continue // Do not put zettel in reload room
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
	ar.nextNum++
	return &anteroom{num: ar.nextNum, next: nil, waiting: waiting, curLoad: 1, reload: false}
}

func (ar *anterooms) Reset() {
	ar.mx.Lock()
	defer ar.mx.Unlock()
	ar.first = ar.makeAnteroom(id.Invalid, arReload)
	ar.last = ar.first
}

func (ar *anterooms) Reload(newZids id.Set) uint64 {
	ar.mx.Lock()
	defer ar.mx.Unlock()
	newWaiting := createWaitingSet(newZids, arUpdate)
	ar.deleteReloadedRooms()

	if ns := len(newWaiting); ns > 0 {
		ar.nextNum++
		ar.first = &anteroom{num: ar.nextNum, next: ar.first, waiting: newWaiting, curLoad: ns}
		if ar.first.next == nil {
			ar.last = ar.first
		}
		return ar.nextNum
	}

	ar.first = nil
	ar.last = nil
	return 0
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

func (ar *anterooms) Dequeue() (arAction, id.Zid, uint64) {
	ar.mx.Lock()
	defer ar.mx.Unlock()
	if ar.first == nil {
		return arNothing, id.Invalid, 0
	}
	for zid, action := range ar.first.waiting {
		roomNo := ar.first.num
		delete(ar.first.waiting, zid)
		if len(ar.first.waiting) == 0 {
			ar.first = ar.first.next
			if ar.first == nil {
				ar.last = nil
			}
		}
		return action, zid, roomNo
	}
	return arNothing, id.Invalid, 0
}
