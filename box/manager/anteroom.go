//-----------------------------------------------------------------------------
// Copyright (c) 2021-present Detlef Stern
//
// This file is part of Zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

package manager

import (
	"sync"

	"zettelstore.de/z/zettel/id"
)

type arAction int

const (
	arNothing arAction = iota
	arReload
	arZettel
)

type anteroom struct {
	num     uint64
	next    *anteroom
	waiting id.Set
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

func newAnterooms(maxLoad int) *anterooms { return &anterooms{maxLoad: maxLoad} }

func (ar *anterooms) EnqueueZettel(zid id.Zid) {
	if !zid.IsValid() {
		return
	}
	ar.mx.Lock()
	defer ar.mx.Unlock()
	if ar.first == nil {
		ar.first = ar.makeAnteroom(zid)
		ar.last = ar.first
		return
	}
	for room := ar.first; room != nil; room = room.next {
		if room.reload {
			continue // Do not put zettel in reload room
		}
		if _, ok := room.waiting[zid]; ok {
			// Zettel is already waiting. Nothing to do.
			return
		}
	}
	if room := ar.last; !room.reload && (ar.maxLoad == 0 || room.curLoad < ar.maxLoad) {
		room.waiting.Zid(zid)
		room.curLoad++
		return
	}
	room := ar.makeAnteroom(zid)
	ar.last.next = room
	ar.last = room
}

func (ar *anterooms) makeAnteroom(zid id.Zid) *anteroom {
	ar.nextNum++
	if zid == id.Invalid {
		return &anteroom{num: ar.nextNum, next: nil, waiting: nil, curLoad: 0, reload: true}
	}
	c := ar.maxLoad
	if c == 0 {
		c = 100
	}
	waiting := id.NewSetCap(ar.maxLoad, zid)
	return &anteroom{num: ar.nextNum, next: nil, waiting: waiting, curLoad: 1, reload: false}
}

func (ar *anterooms) Reset() {
	ar.mx.Lock()
	defer ar.mx.Unlock()
	ar.first = ar.makeAnteroom(id.Invalid)
	ar.last = ar.first
}

func (ar *anterooms) Reload(newZids id.Set) uint64 {
	ar.mx.Lock()
	defer ar.mx.Unlock()
	ar.deleteReloadedRooms()

	if ns := len(newZids); ns > 0 {
		ar.nextNum++
		ar.first = &anteroom{num: ar.nextNum, next: ar.first, waiting: newZids, curLoad: ns, reload: true}
		if ar.first.next == nil {
			ar.last = ar.first
		}
		return ar.nextNum
	}

	ar.first = nil
	ar.last = nil
	return 0
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
	roomNo := ar.first.num
	if ar.first.waiting == nil {
		ar.removeFirst()
		return arReload, id.Invalid, roomNo
	}
	for zid := range ar.first.waiting {
		delete(ar.first.waiting, zid)
		if len(ar.first.waiting) == 0 {
			ar.removeFirst()
		}
		return arZettel, zid, roomNo
	}
	ar.removeFirst()
	return arNothing, id.Invalid, 0
}

func (ar *anterooms) removeFirst() {
	ar.first = ar.first.next
	if ar.first == nil {
		ar.last = nil
	}
}
